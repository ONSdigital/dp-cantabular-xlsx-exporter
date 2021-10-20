package handler

//!!! adjust all of this to read csv and output xlsx
// !!! and also get metadata and put that into the xlsx as part of the above xlsx production
import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	//	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
	//	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	//	"github.com/ONSdigital/dp-api-clients-go/v2/headers"
	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	_ "github.com/aws/aws-sdk-go-v2/aws"
)

// !!! the below needs renaming to suit this service - see what dp-dataset-exporter-xlsx names things and copy
// CsvComplete is the handle for the CsvHandler event
type CsvComplete struct {
	cfg config.Config
	//	ctblr       CantabularClient
	//	datasets    DatasetAPIClient
	s3          S3Uploader
	vaultClient VaultClient
	producer    kafka.IProducer
	generator   Generator
}

// NewCsvComplete creates a new CsvHandler
func NewCsvComplete(cfg config.Config /*c CantabularClient, d DatasetAPIClient,*/, s S3Uploader, v VaultClient, p kafka.IProducer, g Generator) *CsvComplete {
	return &CsvComplete{
		cfg: cfg,
		//		ctblr:       c,
		//		datasets:    d,
		s3:          s,
		vaultClient: v,
		producer:    p,
		generator:   g,
	}
}

// GetS3Downloader creates an S3 Uploader, or a local storage client if a non-empty LocalObjectStore is provided
var GetS3Downloader = func(cfg *config.Config) (*s3manager.Downloader, error) {
	if cfg.LocalObjectStore != "" {
		s3Config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
			Endpoint:         aws.String(cfg.LocalObjectStore),
			Region:           aws.String(cfg.AWSRegion),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}

		// !!! may need to save the session from 'GetS3Uploader' and re-use it here
		sess, err := session.NewSession(s3Config)
		if err != nil {
			//!!! should the following actually say (as in download-service): "could not create the local-object-store s3 client: %w", err   ???
			return nil, fmt.Errorf("failed to create aws session: %w", err)
		}
		return s3manager.NewDownloader(sess), nil
	}

	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})

	return s3manager.NewDownloader(sess), nil
}

// Create a fake WriterAt to wrap a Writer
// from: https://stackoverflow.com/questions/60034007/is-there-an-aws-s3-go-api-for-reading-file-instead-of-download-file
// see also: https://dev.to/flowup/using-io-reader-io-writer-in-go-to-stream-data-3i7b
type FakeWriterAt struct {
	w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	// ignore 'offset' because we forced sequential downloads
	return fw.w.Write(p)
}

// Handle takes a single event.
func (h *CsvComplete) Handle(ctx context.Context, e *event.CommonOutputCreated) error {
	logData := log.Data{
		"event": e,
	}
	log.Info(ctx, "Info from incomming event: CommonOutputCreated :", logData) //!!! for development, trash later

	if e.InstanceID == "" {
		return &Error{
			err:     fmt.Errorf("instanceID is empty"),
			logData: logData,
		}
	}
	filenameCsv := generateS3FilenameCsv(e.InstanceID)
	filenameXlsx := generateS3FilenameXlsx(e.instanceID)

	bucketName := h.s3.BucketName()

	//S3Uploader

	// Create an io.Pipe to have the ability to read what is written to a writer
	csvReader, csvWriter := io.Pipe()

	//var downloader *s3manager.Downloader
	/*	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(h.cfg.AWSRegion))
		if err != nil {
			return &Error{
				err:     fmt.Errorf("aws config error"),
				logData: logData,
			}
		}

		client := s3v2.NewFromConfig(cfg)*/

	// optimize with sync pools, see this article:
	// https://levyeran.medium.com/high-memory-allocations-and-gc-cycles-while-downloading-large-s3-objects-using-the-aws-sdk-for-go-e776a136c5d0

	/*client, err := configS3(h.cfg.MinioAccessKey, h.cfg.MinioSecretKey, bucketName)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("client problem"),
			logData: logData,
		}
	}

	downloader := s3manager.NewDownloader(client)*/
	downloader, err := GetS3Downloader(&h.cfg)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("client problem"),
			logData: logData,
		}
	}
	// Set concurrency to one so the download will be sequential
	downloader.Concurrency = 1

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Wrap the writer created with io.Pipe() with the FakeWriterAt created in the first step.
		// Use the Download function to write to the wrapped Writer:
		numberOfBytesRead, err := downloader.Download( /*context.TODO(),*/ FakeWriterAt{csvWriter},
			&s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(filenameCsv),
			})
		if err != nil {
			//!!! eventually log an error instead of the Info
			log.Info(ctx, "problem downloading unencrypted file from S3", log.Data{"err": err, "bucketName": bucketName, "filenameCsv": filenameCsv})
			csvWriter.CloseWithError(fmt.Errorf("problem downloading unencrypted file from S3"))
		} else {
			//!!! my want to log the following for initial development and then trash displaying this info
			log.Info(ctx, fmt.Sprintf("file downloaded, %d bytes\n", numberOfBytesRead))
			csvWriter.Close()
		}
	}()

	// see  https://www.socketloop.com/tutorials/golang-download-file-example

	/*	req, err := h.s3.GetObject(&s3.GetObjectInput{
			Bucket: aws.String("my.s3.bucket"),
			Key:    aws.String("TEST/test.log"),
		})
		if err != nil {
			panic(err)
		}
		var reader io.Reader
		reader = req.Body*/
	//	data, _ := ioutil.ReadAll(r /*reader*/) // !!! this needs to be read a line at a time ...
	//	var stringData string
	//	stringData = string(data[:])
	scanner := bufio.NewScanner(csvReader /*strings.NewReader(stringData)*/)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		log.Info(ctx, line) //!!! for development, trash later
	}
	wg.Wait()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed scanning csv lines: %s", err)
	}

	// !!! set up a different io.pipe
	// set up another go routine to do the upload to S3 reading from the pipe
	// inside the above for scanner loop do the execell stuff for writing the csv lines.

	// !!! then need to figure out need for encryption of files ?

	// Convert Cantabular Response To CSV file
	/*	file, numBytes, err := h.ParseQueryResponse(resp)
		if err != nil {
			return fmt.Errorf("failed to generate table from query response: %w", err)
		}

		isPublished := true

		// Upload CSV file to S3, note that the S3 file location is ignored
		// because we will use download service to access the file
		_, err = h.UploadCSVFile(ctx, e.InstanceID, file, isPublished)
		if err != nil {
			return &Error{
				err: fmt.Errorf("failed to upload .csv file to S3 bucket: %w", err),
				logData: log.Data{
					"bucket":      h.s3.BucketName(),
					"instance_id": e.InstanceID,
				},
			}
		}*/

	//!!! hmm, may need dataset stuff to go updating instance ??? - ask others about this
	// Update instance with link to file
	/*	if err := h.UpdateInstance(ctx, e.InstanceID, numBytes); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}*/

	//!!! fix following for xlsx
	log.Event(ctx, "producing common output created event", log.INFO, log.Data{})

	//!!! fix following for xlsx output
	//!!! need to figure out what to produce ... or do whatever the dp-dataset-exporter-xlsx does ...
	// Generate output kafka message
	if err := h.ProduceExportCompleteEvent(e.InstanceID); err != nil {
		return fmt.Errorf("failed to produce export complete kafka message: %w", err)
	}
	return nil
}

// ValidateInstance validates the instance returned from dp-dataset-api
func (h *CsvComplete) ValidateInstance(i dataset.Instance) error {
	if len(i.CSVHeader) < 2 {
		return &Error{
			err: errors.New("no dimensions in headers"),
			logData: log.Data{
				"headers": i.CSVHeader,
			},
		}
	}

	return nil
}

// ValidateQueryResponse validates the query response returned from Cantabular:
// - Is not nil
// - Contains at least one dimension
// - Each dimension count matches the number of categories for that dimension
// - Each dimension variable contains a non-empty label
// - Each dimension category contains a non-empty label
// - The total number of values corresponds to all the permutations of possible dimension categories
/*func (h *InstanceComplete) ValidateQueryResponse(resp *cantabular.StaticDatasetQuery) error {
	if resp == nil {
		return errors.New("nil response")
	}
	if len(resp.Dataset.Table.Dimensions) == 0 {
		return errors.New("no dimension in response")
	}

	expectedNumValues := 1
	for _, dim := range resp.Dataset.Table.Dimensions {
		expectedNumValues *= dim.Count

		if dim.Variable.Label == "" {
			return errors.New("empty variable label in cantabular response")
		}

		if len(dim.Categories) != dim.Count {
			return NewError(
				errors.New("wrong number of categories for a dimensions in response"),
				log.Data{
					"dimension":         dim.Variable.Label,
					"dimension_count":   dim.Count,
					"categories_length": len(dim.Categories),
				},
			)
		}

		for _, category := range dim.Categories {
			if category.Label == "" {
				return errors.New("empty category label in cantabular response")
			}
		}
	}

	if len(resp.Dataset.Table.Values) != expectedNumValues {
		return NewError(
			errors.New("wrong number of values in response"),
			log.Data{
				"expected_values": expectedNumValues,
				"values_length":   len(resp.Dataset.Table.Values),
			},
		)
	}

	return nil
}

// ParseQueryResponse parses the provided cantabular response into a CSV bufio.Reader,
// where the first row corresponds to the dimension names header (including a count)
// and each subsequent row corresponds to a unique combination of dimension values and their count.
//
// Example by Sensible Code here: https://github.com/cantabular/examples/blob/master/golang/main.go
func (h *InstanceComplete) ParseQueryResponse(resp *cantabular.StaticDatasetQuery) (*bufio.Reader, int, error) {
	// Create CSV writer with underlying buffer
	b := new(bytes.Buffer)
	w := csv.NewWriter(b)

	// aux func to write to the csv writer, returning any error (returned by w.Write or w.Error)
	write := func(record []string) error {
		if err := w.Write(record); err != nil {
			return err
		}
		return w.Error()
	}

	// Obtain the CSV header
	header := createCSVHeader(resp.Dataset.Table.Dimensions)
	if err := write(header); err != nil {
		return nil, 0, fmt.Errorf("error writing the csv header: %w", err)
	}

	// Obtain the CSV rows according to the cantabular dimensions and counts
	for i, count := range resp.Dataset.Table.Values {
		row := createCSVRow(resp.Dataset.Table.Dimensions, i, count)
		if err := write(row); err != nil {
			return nil, 0, fmt.Errorf("error writing a csv row: %w", err)
		}
	}

	// Flush to make sure all data is present in the byte buffer
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, 0, fmt.Errorf("error flushing the csv writer: %w", err)
	}

	// Return a reader with the same underlying Byte buffer that is written by the csv writter
	return bufio.NewReader(b), b.Len(), nil
}

// createCSVHeader creates an array of strings corresponding to a csv header
// where each column contains the value of the corresponding dimension, with the last column being the 'count'
func createCSVHeader(dims []cantabular.Dimension) []string {
	header := make([]string, len(dims)+1)
	for i, dim := range dims {
		header[i+1] = dim.Variable.Label
	}
	header[0] = "cantabular_blob"
	return header
}

// createCSVRow creates an array of strings corresponding to a csv row
// for the provided array of dimension, index and count
// it assumes that the values are sorted with lower weight for the last dimension and higher weight for the first dimension.
func createCSVRow(dims []cantabular.Dimension, index, count int) []string {
	row := make([]string, len(dims)+1)
	// Iterate dimensions starting from the last one (lower weight)
	for i := len(dims) - 1; i >= 0; i-- {
		catIndex := index % dims[i].Count             // Index of the category for the current dimension
		row[i+1] = dims[i].Categories[catIndex].Label // The CSV column corresponds to the label of the Category
		index /= dims[i].Count                        // Modify index for next iteration
	}
	row[0] = fmt.Sprintf("%d", count)
	return row
}

// UploadCSVFile uploads the provided file content to AWS S3
func (h *InstanceComplete) UploadCSVFile(ctx context.Context, instanceID string, file io.Reader, isPublished bool) (string, error) {
	if instanceID == "" {
		return "", errors.New("empty instance id not allowed")
	}
	if file == nil {
		return "", errors.New("no file content has been provided")
	}

	bucketName := h.s3.BucketName()
	filename := generateS3Filename(instanceID)

	// As the code is now it is assumed that the file is always published
	if isPublished {

		logData := log.Data{
			"bucket":       bucketName,
			"filename":     filename,
			"is_published": true,
		}

		log.Info(ctx, "uploading published file to S3", logData)

		result, err := h.s3.Upload(&s3manager.UploadInput{
			Body:   file,
			Bucket: &bucketName,
			Key:    &filename,
		})
		if err != nil {
			return "", NewError(
				fmt.Errorf("failed to upload published file to S3: %w", err),
				logData,
			)
		}

		return url.PathUnescape(result.Location)

	}

	logData := log.Data{
		"bucket":              bucketName,
		"filename":            filename,
		"encryption_disabled": h.cfg.EncryptionDisabled,
		"is_published":        false,
	}

	if h.cfg.EncryptionDisabled {
		log.Info(ctx, "uploading unencrypted file to S3", logData)

		result, err := h.s3.Upload(&s3manager.UploadInput{
			Body:   file,
			Bucket: &bucketName,
			Key:    &filename,
		})
		if err != nil {
			return "", NewError(
				fmt.Errorf("failed to upload unencrypted file to S3: %w", err),
				logData,
			)
		}

		return url.PathUnescape(result.Location)
	}

	log.Info(ctx, "uploading encrypted file to S3", logData)

	psk, err := h.generator.NewPSK()
	if err != nil {
		return "", NewError(
			fmt.Errorf("failed to generate a PSK for encryption: %w", err),
			logData,
		)
	}

	vaultPath := generateVaultPathForFile(h.cfg.VaultPath, instanceID)
	vaultKey := "key"

	log.Info(ctx, "writing key to vault", log.Data{"vault_path": vaultPath})

	if err := h.vaultClient.WriteKey(vaultPath, vaultKey, hex.EncodeToString(psk)); err != nil {
		return "", NewError(
			fmt.Errorf("failed to write key to vault: %w", err),
			logData,
		)
	}

	result, err := h.s3.UploadWithPSK(&s3manager.UploadInput{
		Body:   file,
		Bucket: &bucketName,
		Key:    &filename,
	}, psk)
	if err != nil {
		return "", NewError(
			fmt.Errorf("failed to upload encrypted file to S3: %w", err),
			logData,
		)
	}

	return url.PathUnescape(result.Location)
}*/

// UpdateInstance updates the instance downlad CSV link using dataset API PUT /instances/{id} endpoint
// note that the URL refers to the download service (it is not the URL returned by the S3 client directly)
/*func (h *InstanceComplete) UpdateInstance(ctx context.Context, instanceID string, size int) error {
	downloadURL := generateURL(h.cfg.DownloadServiceURL, instanceID)
	update := dataset.UpdateInstance{
		Downloads: dataset.DownloadList{
			CSV: &dataset.Download{
				URL:  downloadURL,             // download service URL for the CSV file
				Size: fmt.Sprintf("%d", size), // size of the file in number of bytes
			},
		},
	}
	if _, err := h.datasets.PutInstance(ctx, "", h.cfg.ServiceAuthToken, "", instanceID, update, headers.IfMatchAnyETag); err != nil {
		return fmt.Errorf("error during put instance: %w", err)
	}
	return nil
}*/

// ProduceExportCompleteEvent sends the final kafka message signifying the export complete
func (h *CsvComplete) ProduceExportCompleteEvent(instanceID string) error {
	downloadURL := generateURL(h.cfg.DownloadServiceURL, instanceID)

	// create CommonOutputCreated event and Marshal it
	b, err := schema.CommonOutputCreated.Marshal(&event.CommonOutputCreated{
		InstanceID: instanceID,
		FileURL:    downloadURL, // download service URL for the CSV file
	})
	if err != nil {
		return fmt.Errorf("error marshalling instance complete event: %w", err)
	}

	// Send bytes to kafka producer output channel
	h.producer.Channels().Output <- b

	return nil
}

// generateURL generates the download service URL for the provided instanceID CSV file
func generateURL(downloadServiceURL, instanceID string) string {
	return fmt.Sprintf("%s/downloads/instances/%s.csv",
		downloadServiceURL,
		instanceID,
	)
}

// generateS3FilenameCsv generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID CSV file that is going to be read
func generateS3FilenameCsv(instanceID string) string {
	return fmt.Sprintf("instances/%s.csv", instanceID)
}

// generateVaultPathForFile generates the vault path for the provided root and filename
func generateVaultPathForFile(vaultPathRoot, instanceID string) string {
	return fmt.Sprintf("%s/%s.csv", vaultPathRoot, instanceID)
}

// generateS3FilenameXlsx generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID XLSX file that is going to be written
func generateS3FilenameXlsx(instanceID string) string {
	return fmt.Sprintf("instances/%s.csv", instanceID)
}
