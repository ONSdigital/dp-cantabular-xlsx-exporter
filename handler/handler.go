package handler

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

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
	"github.com/xuri/excelize/v2"
)

const (
	maxObservationCount = 999900 //!!! the name of this might be wrong ?
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
			return nil, fmt.Errorf("failed to create aws session (local): %w", err)
		}
		return s3manager.NewDownloader(sess), nil //!!! ultimatley this needs to be more like the csv-exporter's GetS3Uploader
	}

	//!!! ultimatley this needs to be more like the csv-exporter's GetS3Uploader, for rest of this function
	//!!! and process any error return
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})

	return s3manager.NewDownloader(sess), nil
}

// FakeWriterAt is a fake WriterAt to wrap a Writer
// from: https://stackoverflow.com/questions/60034007/is-there-an-aws-s3-go-api-for-reading-file-instead-of-download-file
// see also: https://dev.to/flowup/using-io-reader-io-writer-in-go-to-stream-data-3i7b
type FakeWriterAt struct {
	w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	// ignore 'offset' because we forced sequential downloads
	_ = offset // stop any linters complaining
	return fw.w.Write(p)
}

// Handle takes a single event.
func (h *CsvComplete) Handle(ctx context.Context, e *event.CantabularCsvCreated) error {
	logData := log.Data{
		"event": e,
	}

	if e.RowCount > maxObservationCount {
		return &Error{
			err:     fmt.Errorf("full download too large to export to .xlsx file"),
			logData: logData,
		}
	}

	if e.InstanceID == "" {
		return &Error{
			err:     fmt.Errorf("instanceID is empty"),
			logData: logData,
		}
	}
	filenameCsv := generateS3FilenameCSV(e.InstanceID)
	filenameXlsx := generateS3FilenameXLSX(e.InstanceID)

	bucketName := h.s3.BucketName()

	// Create an io.Pipe to have the ability to read what is written to a writer
	csvReader, csvWriter := io.Pipe()

	// optimize with sync pools, see this article:
	// https://levyeran.medium.com/high-memory-allocations-and-gc-cycles-while-downloading-large-s3-objects-using-the-aws-sdk-for-go-e776a136c5d0

	// !!! get the metadata

	// start creating the excel file
	excelFile := excelize.NewFile()
	streamWriter, err := excelFile.NewStreamWriter("Dataset")
	if err != nil {
		return &Error{
			err:     fmt.Errorf("excel stream writer creation problem"),
			logData: logData,
		}
	}

	// !!! write header on first sheet, just to demonstrate ... (if its to be kept, add error handling)
	styleID, err := excelFile.NewStyle(`{"font":{"color":"#EE2277"}}`)
	if err != nil {
		fmt.Println(err)
	}
	if err := streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{StyleID: styleID, Value: "Data"}}); err != nil {
		fmt.Println(err)
	}
	// !!! above section for test & demonstration only

	// !!! need to figure out what to do about not yet published files ... (that is encrypted incoming CSV)
	downloader, err := GetS3Downloader(&h.cfg)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("downloader client problem"),
			logData: logData,
		}
	}

	// Set concurrency to one so the download will be sequential (which is essential to stream reading file in order)
	downloader.Concurrency = 1

	downloadCtx, cancelDownload := context.WithCancel(ctx)
	defer cancelDownload()

	wgDownload := sync.WaitGroup{}
	wgDownload.Add(1)
	go func(ctx context.Context) {
		defer wgDownload.Done()

		// Wrap the writer created with io.Pipe() with the FakeWriterAt created in the first step.
		// Use the DownloadWithContext function to write to the wrapped Writer:
		numberOfBytesRead, err := downloader.DownloadWithContext(ctx, FakeWriterAt{csvWriter},
			&s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(filenameCsv),
			})
		if err != nil {
			report := &Error{
				err:     err,
				logData: log.Data{"err": err, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}

			if closeErr := csvWriter.CloseWithError(report); closeErr != nil {
				log.Error(ctx, "error closing download writerWithError", closeErr)
			}
		} else {
			log.Info(ctx, fmt.Sprintf(".csv file: %s, downloaded from bucket: %s, length: %d bytes", filenameCsv, bucketName, numberOfBytesRead))

			if closeErr := csvWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing download writer", closeErr)
			}
		}
	}(downloadCtx)

	var outputRow = 3 // !!! this value choosen for test to visually see effect in excel spreadsheet AND most importantly to NOT touch any cells previously streamed to above in test code

	var incomingCsvRow = 0
	scanner := bufio.NewScanner(csvReader)
	for scanner.Scan() {
		incomingCsvRow++
		line := scanner.Text()

		// split 'line' and do the excel stream write at 'row' & deal with any errors
		columns := strings.Split(line, ",")
		nofColumns := len(columns)
		if nofColumns == 0 {
			return &Error{
				err:     fmt.Errorf("downloaded .csv file has no columns at row %d", incomingCsvRow),
				logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}
		rowItems := make([]interface{}, nofColumns)
		for colID := 0; colID < nofColumns; colID++ {
			rowItems[colID] = columns[colID]
		}
		cell, err := excelize.CoordinatesToCellName(1, outputRow)
		if err != nil {
			return &Error{
				err:     err,
				logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
			}
		}
		if err := streamWriter.SetRow(cell, rowItems); err != nil {
			return &Error{
				err:     err,
				logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
			}
		}
		outputRow++
	}
	if err := scanner.Err(); err != nil {
		return &Error{
			err:     err,
			logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow, "Bingo": "*** wow ***"},
		}
	}

	// All of the CSV file has now been downloaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgDownload.Wait()

	// Must now finish up the CSV excelize streamWriter before doing excelize API calls in building up metadata sheet:
	if err := streamWriter.Flush(); err != nil {
		return &Error{
			err:     err,
			logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv},
		}
	}

	//!!! add in the metadata to sheet 2, and deal with any errors
	// -=-=- : example test code for demo, using the excelize API calls ONLY (no more streaming) ...
	excelFile.NewSheet("Metadata")
	// Set value of a cell.
	excelFile.SetCellValue("Metadata", "A1", "Place")
	excelFile.SetCellValue("Metadata", "B1", "Metadata")
	excelFile.SetCellValue("Metadata", "C1", "here ...")

	excelFile.SetCellValue("Metadata", "B9", "Hello")
	excelFile.SetCellValue("Metadata", "C10", "world")

	// Set active sheet of the workbook.
	excelFile.SetActiveSheet(excelFile.GetSheetIndex("Dataset"))
	// -=-=-

	xlsxReader, xlsxWriter := io.Pipe()

	wgUpload := sync.WaitGroup{}
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()

		// Write the 'in memory' spreadsheet to the given io.writer
		if err := excelFile.Write(xlsxWriter); err != nil {
			report := &Error{
				err:     err,
				logData: log.Data{"err": err, "bucketName": bucketName, "filenameXlsx": filenameXlsx},
			}

			if closeErr := xlsxWriter.CloseWithError(report); closeErr != nil {
				log.Error(ctx, "error closing upload writerWithError", closeErr)
			}
		} else {
			log.Info(ctx, fmt.Sprintf(".xlsx file: %s, finished writing to pipe for bucket: %s", filenameXlsx, bucketName)) // !!! do we want this log line or the one "XLSX file uploaded to" further on ?

			if closeErr := xlsxWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing upload writer", closeErr)
			}
		}
	}()

	isPublished := true

	// Use the Upload function to read from the io.Pipe() Writer:
	_, err = h.UploadXLSXFile(ctx, e.InstanceID, xlsxReader, isPublished)
	if err != nil {
		return &Error{
			err: fmt.Errorf("failed to upload .xlsx file to S3 bucket: %w", err),
			logData: log.Data{
				"bucket":      h.s3.BucketName(),
				"instance_id": e.InstanceID,
			},
		}
	}

	// All of the CSV file has now been uploaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgUpload.Wait()

	// !!! then need to figure out need for encryption of files ?

	// Convert Cantabular Response To CSV file
	/*	file, numBytes, err := h.ParseQueryResponse(resp)
		if err != nil {
			return fmt.Errorf("failed to generate table from query response: %w", err)
		}

		isPublished := true

		// Upload CSV file to S3, note that the S3 file location is ignored
		// because we will use download service to access the file !!! this comment seems wrong because the donwload service downloads and the following code is doing an upload !!!
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

// UploadXLSXFile uploads the provided file content to AWS S3
func (h *CsvComplete) UploadXLSXFile(ctx context.Context, instanceID string, file io.Reader, isPublished bool) (string, error) {
	if instanceID == "" {
		return "", errors.New("empty instance id not allowed")
	}
	if file == nil {
		return "", errors.New("no file content has been provided")
	}

	bucketName := h.s3.BucketName() //!!! this is flawed as there should be seperate 'S3PrivateBucketName' and 'S3BucketName'
	filename := generateS3FilenameXLSX(instanceID)

	// As the code is now it is assumed that the file is always published - TODO, thi function needs rationalising once full system is in place
	if isPublished {

		logData := log.Data{
			"bucket":       bucketName,
			"filename":     filename,
			"is_published": true,
		}

		log.Info(ctx, "uploading published file to S3", logData)

		// !!! this code needs to use 'UploadWithContext' ???, because when processing an excel file that is
		// nearly 1million lines it has been seen to take over 45 seconds and if nomad has instructed a service
		// to shut down gracefully before installing a new version of this app, then this could cause problems.
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

		// !!! this code needs to use 'UploadWithContext' ???, because when processing an excel file that is
		// nearly 1million lines it has been seen to take over 45 seconds and if nomad has instructed a service
		// to shut down gracefully before installing a new version of this app, then this could cause problems.
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

	// !!! this code needs to use 'UploadWithContext' ???, because when processing an excel file that is
	// nearly 1million lines it has been seen to take over 45 seconds and if nomad has instructed a service
	// to shut down gracefully before installing a new version of this app, then this could cause problems.
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
}

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

//!!! need to have discussion to determine what the output of this service should be
// ProduceExportCompleteEvent sends the final kafka message signifying the export complete
func (h *CsvComplete) ProduceExportCompleteEvent(instanceID string) error {
	//!!!	downloadURL := generateURL(h.cfg.DownloadServiceURL, instanceID)

	// create InstanceComplete event and Marshal it
	b, err := schema.InstanceComplete.Marshal(&event.InstanceComplete{
		InstanceID: instanceID,
		//!!!		FileURL:    downloadURL, // download service URL for the CSV file
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

// generateS3FilenameCSV generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID CSV file that is going to be read
func generateS3FilenameCSV(instanceID string) string {
	return fmt.Sprintf("instances/%s.csv", instanceID)
	//return fmt.Sprintf("instances/1000Kx50.csv")
	//return fmt.Sprintf("instances/50Kx50.csv")
}

// generateVaultPathForFile generates the vault path for the provided root and filename
func generateVaultPathForFile(vaultPathRoot, instanceID string) string {
	return fmt.Sprintf("%s/%s.csv", vaultPathRoot, instanceID)
}

// generateS3FilenameXLSX generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID XLSX file that is going to be written
func generateS3FilenameXLSX(instanceID string) string {
	return fmt.Sprintf("instances/%s.xlsx", instanceID)
}
