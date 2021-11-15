package handler

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/ONSdigital/dp-api-clients-go/headers"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
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
	maxAllowedRowCount       = 999900
	smallEnoughForFullFormat = 10000 // Not too large to achieve full formatting in memory
)

// !!! the below needs renaming to suit this service - see what dp-dataset-exporter-xlsx names things and copy
// CsvComplete is the handle for the CsvHandler event
type CsvComplete struct {
	cfg         config.Config
	datasets    DatasetAPIClient
	s3Private   S3Uploader
	s3Public    S3Uploader
	vaultClient VaultClient
	producer    kafka.IProducer
	generator   Generator
}

// NewCsvComplete creates a new CsvHandler
func NewCsvComplete(cfg config.Config, d DatasetAPIClient, sPrivate S3Uploader, sPublic S3Uploader, v VaultClient, p kafka.IProducer, g Generator) *CsvComplete {
	return &CsvComplete{
		cfg:         cfg,
		datasets:    d,
		s3Private:   sPrivate,
		s3Public:    sPublic,
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

	if e.RowCount > maxAllowedRowCount {
		// !!! change this to a log info and also report that job complete with no result due to too large an CSV file
		return &Error{err: fmt.Errorf("full download too large to export to .xlsx file"),
			logData: logData,
		}
	}

	if e.InstanceID == "" {
		return &Error{err: fmt.Errorf("instanceID is empty"),
			logData: logData,
		}
	}

	instance, _, err := h.datasets.GetInstance(ctx, "", h.cfg.ServiceAuthToken, "", e.InstanceID, headers.IfMatchAnyETag)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("failed to get instance: %w", err),
			logData: logData,
		}
	}

	log.Info(ctx, "instance obtained from dataset API", log.Data{
		"instance_id":    instance.ID,
		"instance_state": instance.State,
	})

	isPublished := instance.State == dataset.StatePublished.String()

	doLargeSheet := true

	if e.RowCount <= smallEnoughForFullFormat {
		// The number of lines in the CSV file is small enough to use the excelize API calls to create
		// an excel file where we can determine and set the column widths to provide the user with
		// as good a presented excel file as we can.
		// (As of Nov 2021 the excelize streaming library code does not allow setting of column widths,
		//  but we use the stream API calls for its speed and memory allocation efficiency for larger
		//  CSV files).
		// NOTE: the excelize libraries use of the word 'stream' is misleading as its actually the
		// excelize libraries efficient mechanism of storring large sheets in memory and has nothing
		// to do with the meaning of the word 'streaming'.
		doLargeSheet = false
	}

	// start creating the excel file in its "in memory structure"
	excelInMemoryStructure := excelize.NewFile()
	sheet1 := "Sheet1"
	var efficientExcelAPIWriter *excelize.StreamWriter
	//var err error
	if doLargeSheet {
		efficientExcelAPIWriter, err = excelInMemoryStructure.NewStreamWriter(sheet1) // have to start with the one and only default 'Sheet1'
		if err != nil {
			return &Error{err: fmt.Errorf("excel stream writer creation problem"),
				logData: logData,
			}
		}
	}

	excelInMemoryStructure.SetDefaultFont("Aerial")

	// !!! write header on first sheet, just to demonstrate ... this may not be needed - TBD
	if err = ApplyMainSheetHeader(excelInMemoryStructure, doLargeSheet, efficientExcelAPIWriter, sheet1); err != nil {
		if err != nil {
			return &Error{err: fmt.Errorf("ApplyMainSheetHeader failed: %w", err),
				logData: logData,
			}
		}
	}

	if err = h.GetCSVtoExcelStructure(ctx, excelInMemoryStructure, e, doLargeSheet, efficientExcelAPIWriter, sheet1, isPublished); err != nil {
		if err != nil {
			return &Error{err: fmt.Errorf("GetCSVtoExcelStructure failed: %w", err),
				logData: logData,
			}
		}
	}

	if err = h.AddMetaDataToExcelStructure(excelInMemoryStructure); err != nil {
		return &Error{err: fmt.Errorf("AddMetaDataToExcelStructure failed: %w", err),
			logData: logData,
		}
	}

	// Rename the main sheet to 'Dataset'
	sheetDataset := "Dataset"
	excelInMemoryStructure.SetSheetName(sheet1, sheetDataset)

	// Set active sheet of the workbook.
	excelInMemoryStructure.SetActiveSheet(excelInMemoryStructure.GetSheetIndex(sheetDataset))

	s3Path, err := h.SaveExcelStructureToExcelFile(ctx, excelInMemoryStructure, e.InstanceID, isPublished)
	if err != nil {
		return &Error{err: fmt.Errorf("SaveExcelStructureToExcelFile failed: %w", err),
			logData: logData,
		}
	}

	// !!! probably need to get size of created file from S3
	var r int = 10000 // !!! temp, for test ... get actual file size
	xlsxDownload := &dataset.Download{
		Size: strconv.Itoa(r),
	}

	if isPublished {
		xlsxDownload.Public = s3Path
	} else {
		xlsxDownload.Private = s3Path
	}

	downloadURL := fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.csv",
		h.cfg.DownloadServiceURL,
		e.DatasetID,
		e.Edition,
		e.Version,
	)

	if err := h.updateVersionLinks(ctx, e, isPublished, xlsxDownload, downloadURL); err != nil {
		return err
	}
	//!!! hmm, may need dataset stuff to go updating instance ??? - ask others about this
	// Update instance with link to file
	/*	if err := h.UpdateInstance(ctx, e.InstanceID, numBytes); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}*/

	//!!! fix following for xlsx
	log.Event(ctx, "producing common output created event", log.INFO, log.Data{"s3Path": s3Path})

	//!!! fix following for xlsx output
	//!!! need to figure out what to produce ... or do whatever the dp-dataset-exporter-xlsx does ...
	// Generate output kafka message
	if err := h.ProduceExportCompleteEvent(e.InstanceID); err != nil {
		return fmt.Errorf("failed to produce export complete kafka message: %w", err)
	}
	return nil
}

// GetCSVtoExcelStructure streams in a line at a time from csv file from S3 bucket and
// inserts it into the excel "in memory structure"
func (h *CsvComplete) GetCSVtoExcelStructure(ctx context.Context, excelInMemoryStructure *excelize.File, e *event.CantabularCsvCreated, doLargeSheet bool, efficientExcelAPIWriter *excelize.StreamWriter, sheet1 string, isPublished bool) error {
	var bucketName string
	if isPublished {
		bucketName = h.s3Public.BucketName()
	} else {
		bucketName = h.s3Private.BucketName()
	}

	downloader, err := GetS3Downloader(&h.cfg)
	if err != nil {
		return err
	}

	// Set concurrency to one so the download will be sequential (which is essential to stream reading file in order)
	downloader.Concurrency = 1

	filenameCsv := generateS3FilenameCSV(e.InstanceID)

	// Create an io.Pipe to have the ability to read what is written to a writer
	csvReader, csvWriter := io.Pipe()

	// !!! need to use 'isPublished' to determine if to get encrypted file ...
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
			report := &Error{err: fmt.Errorf("DownloadWithContext failed, %w", err),
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

	var startRow = 3
	var outputRow = startRow // !!! this value choosen for test to visually see effect in excel spreadsheet
	// AND most importantly to NOT touch any cells previously created with the excelize streamWriter mechanism

	var maxCol = 1

	styleID14, err := excelInMemoryStructure.NewStyle(`{"font":{"size":14}}`)
	if err != nil {
		return &Error{err: fmt.Errorf("NewStyle size 14 %w", err)}
	}
	var incomingCsvRow = 0
	scanner := bufio.NewScanner(csvReader)
	for scanner.Scan() {
		incomingCsvRow++
		line := scanner.Text()

		// split 'line' and do the excel write at 'row' & deal with any errors
		columns := strings.Split(line, ",")
		nofColumns := len(columns)
		if nofColumns == 0 {
			return &Error{err: fmt.Errorf("downloaded .csv file has no columns at row %d", incomingCsvRow),
				logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}
		if nofColumns > maxCol {
			maxCol = nofColumns
		}
		var rowItemsWithStyle []interface{}
		for colID := 0; colID < nofColumns; colID++ {
			value := columns[colID]
			valueFloat, err := strconv.ParseFloat(value, 64)
			if doLargeSheet {
				if err == nil {
					rowItemsWithStyle = append(rowItemsWithStyle, excelize.Cell{StyleID: styleID14, Value: valueFloat})
				} else {
					rowItemsWithStyle = append(rowItemsWithStyle, excelize.Cell{StyleID: styleID14, Value: value})
				}
			} else {
				if err == nil {
					rowItemsWithStyle = append(rowItemsWithStyle, valueFloat)
				} else {
					rowItemsWithStyle = append(rowItemsWithStyle, value)
					//!!! need to gather the max width of each column for all rows (clamping max value to 255 for the excelize library limit of 255)
				}
			}
		}

		if doLargeSheet {
			cell, err := excelize.CoordinatesToCellName(1, outputRow)
			if err != nil {
				return &Error{err: err,
					logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
			if err := efficientExcelAPIWriter.SetRow(cell, rowItemsWithStyle); err != nil {
				return &Error{err: err,
					logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
		} else {
			addr, err := excelize.JoinCellName("A", outputRow)
			if err != nil {
				return &Error{err: fmt.Errorf("JoinCellName %w", err)}
			}
			if err := excelInMemoryStructure.SetSheetRow(sheet1, addr, &rowItemsWithStyle); err != nil {
				return &Error{err: fmt.Errorf("SetSheetRow 2 %w", err),
					logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
		}
		outputRow++
	}
	if err := scanner.Err(); err != nil {
		return &Error{err: err,
			// !!! remove Bingo at some point
			logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow, "Bingo": "*** wow ***"},
		}
	}

	// All of the CSV file has now been downloaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgDownload.Wait()

	if doLargeSheet {
		// Must now finish up the CSV excelize streamWriter calls before doing excelize API calls in building up metadata sheet:
		if err := efficientExcelAPIWriter.Flush(); err != nil {
			return &Error{err: err,
				logData: log.Data{"event": e, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}
	} else {
		// set font style for range of cells written
		if err = ApplySmallSheetCellStyle(excelInMemoryStructure, startRow, maxCol, outputRow, sheet1, styleID14); err != nil {
			return &Error{err: fmt.Errorf("ApplySmallSheetCellStyle %w", err)}
		}

		err = excelInMemoryStructure.SetColWidth(sheet1, "A", "B", 24) //!!! this is for test and needs further work to apply desired widths for all columns
		if err != nil {
			return &Error{err: err}
		}
		err = excelInMemoryStructure.SetColWidth(sheet1, "C", "C", 40) //!!! this is for test and needs further work to apply desired widths for all columns
		if err != nil {
			return &Error{err: err}
		}
	}

	return nil
}

// SaveExcelStructureToExcelFile uses the excelize library Write function to effectively write out the excel
// "in memory structure" to a stream that is then streamed directly into a file in S3 bucket.
// returns s3Location (path) or Error
func (h *CsvComplete) SaveExcelStructureToExcelFile(ctx context.Context, excelInMemoryStructure *excelize.File, instanceID string, isPublished bool) (string, error) {
	var bucketName string
	if isPublished {
		bucketName = h.s3Public.BucketName()
	} else {
		bucketName = h.s3Private.BucketName()
	}

	filenameXlsx := generateS3FilenameXLSX(instanceID)
	xlsxReader, xlsxWriter := io.Pipe()

	wgUpload := sync.WaitGroup{}
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()

		// Write the 'in memory' spreadsheet to the given io.writer
		if err := excelInMemoryStructure.Write(xlsxWriter); err != nil {
			report := &Error{err: err,
				logData: log.Data{"err": err, "bucketName": bucketName, "filenameXlsx": filenameXlsx},
			}

			if closeErr := xlsxWriter.CloseWithError(report); closeErr != nil {
				log.Error(ctx, "error closing upload writerWithError", closeErr)
			}
		} else {
			log.Info(ctx, fmt.Sprintf("finished writing file: %s, to pipe for bucket: %s", filenameXlsx, bucketName)) // !!! do we want this log line or the one "XLSX file uploaded to" further on ?

			if closeErr := xlsxWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing upload writer", closeErr)
			}
		}
	}()

	// Use the Upload function to read from the io.Pipe() Writer:
	s3Path, err := h.UploadXLSXFile(ctx, instanceID, xlsxReader, isPublished, bucketName, filenameXlsx)
	if err != nil {
		if closeErr := xlsxWriter.Close(); closeErr != nil {
			log.Error(ctx, "error closing upload writer", closeErr)
		}

		return "", &Error{err: fmt.Errorf("failed to upload .xlsx file to S3 bucket: %w", err),
			logData: log.Data{
				"bucket":      bucketName,
				"instance_id": instanceID,
			},
		}
	}

	// All of the XLSX file has now been uploaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgUpload.Wait()

	return s3Path, nil
}

// UploadXLSXFile uploads the provided file content to AWS S3
// returns s3Location (path) or Error
func (h *CsvComplete) UploadXLSXFile(ctx context.Context, instanceID string, file io.Reader, isPublished bool, bucketName string, filename string) (string, error) {
	if instanceID == "" {
		return "", errors.New("empty instance id not allowed")
	}
	if file == nil {
		return "", errors.New("no file content has been provided")
	}

	resultPath := ""
	logData := log.Data{
		"bucket":              bucketName,
		"filename":            filename,
		"encryption_disabled": h.cfg.EncryptionDisabled,
		"is_published":        isPublished,
	}

	// As the code is now it is assumed that the file is always published - TODO, this function needs rationalising once full system is in place
	if isPublished {
		log.Info(ctx, "uploading published file to S3", logData)

		// We use UploadWithContext because when processing an excel file that is
		// nearly 1million lines it has been seen to take over 45 seconds and if nomad has instructed a service
		// to shut down gracefully before installing a new version of this app, then this could cause problems.
		result, err := h.s3Public.UploadWithContext(ctx, &s3manager.UploadInput{
			Body:   file,
			Bucket: &bucketName,
			Key:    &filename,
		})
		if err != nil {
			return "", NewError(fmt.Errorf("UploadWithContext failed to upload published file to S3: %w", err),
				logData,
			)
		}
		resultPath = result.Location
	} else {
		logData := log.Data{
			"encryption_disabled": h.cfg.EncryptionDisabled,
		}
		if h.cfg.EncryptionDisabled {
			log.Info(ctx, "uploading unencrypted file to S3", logData)

			result, err := h.s3Private.UploadWithContext(ctx, &s3manager.UploadInput{
				Body:   file,
				Bucket: &bucketName,
				Key:    &filename,
			})
			if err != nil {
				return "", NewError(fmt.Errorf("UploadWithContext failed to upload unencrypted file to S3: %w", err),
					logData,
				)
			}
			resultPath = result.Location
		} else {
			log.Info(ctx, "uploading encrypted file to S3", logData)

			psk, err := h.generator.NewPSK()
			if err != nil {
				return "", NewError(fmt.Errorf("NewPSK failed to generate a PSK for encryption: %w", err),
					logData,
				)
			}

			vaultPath := generateVaultPathForFile(h.cfg.VaultPath, instanceID)
			vaultKey := "key"

			log.Info(ctx, "writing key to vault", log.Data{"vault_path": vaultPath})

			if err := h.vaultClient.WriteKey(vaultPath, vaultKey, hex.EncodeToString(psk)); err != nil {
				return "", NewError(fmt.Errorf("WriteKey failed to write key to vault: %w", err),
					logData,
				)
			}

			// !!! this code needs to use 'UploadWithContextPSK' ???, because when processing an excel file that is
			// nearly 1 million lines it has been seen to take over 45 seconds and if nomad has instructed a service
			// to shut down gracefully before installing a new version of this app, then this could cause problems.
			result, err := h.s3Private.UploadWithPSK(&s3manager.UploadInput{
				Body:   file,
				Bucket: &bucketName,
				Key:    &filename,
			}, psk)
			if err != nil {
				return "", NewError(fmt.Errorf("UploadWithPSK failed to upload encrypted file to S3: %w", err),
					logData,
				)
			}
			resultPath = result.Location
		}
	}

	s3Location, err := url.PathUnescape(resultPath)
	if err != nil {
		logData["location"] = resultPath
		return "", NewError(fmt.Errorf("failed to unescape S3 path location: %w", err),
			logData,
		)
	}
	return s3Location, nil
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

func (h *CsvComplete) updateVersionLinks(ctx context.Context, event *event.CantabularCsvCreated, isPublished bool, xlsx *dataset.Download, downloadURL string) error {
	xlsx.URL = downloadURL // + metadataExtension !!! sus whth this needs to be if anything

	log.Info(ctx, "updating dataset api with download link", log.Data{
		"isPublished":  isPublished,
		"xlsxDownload": xlsx,
	})

	v := dataset.Version{
		Downloads: map[string]dataset.Download{
			"XLSX": *xlsx,
		},
	}

	err := h.datasets.PutVersion(
		ctx, "", h.cfg.ServiceAuthToken, "", event.DatasetID, event.Edition, event.Version, v)
	if err != nil {
		return &Error{err: fmt.Errorf("error while attempting update version downloads: %w", err)}
	}

	return nil
}

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
	// return fmt.Sprintf("instances/1000Kx50.csv")//!!! for non stream code this crashes using 13GB RAM in docker
	// return fmt.Sprintf("instances/50Kx50.csv") //!!! this uses 1.7GB for non large excel code
	// return fmt.Sprintf("instances/10Kx7.csv")
	// return fmt.Sprintf("instances/25Kx7.csv")
	// return fmt.Sprintf("instances/50Kx7.csv")
	// return fmt.Sprintf("instances/100Kx7.csv")
	// return fmt.Sprintf("instances/1000Kx7.csv")
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

// ApplyMainSheetHeader puts relevant header information in first rows of sheet
func ApplyMainSheetHeader(excelInMemoryStructure *excelize.File, doLargeSheet bool, efficientExcelAPIWriter *excelize.StreamWriter, sheet1 string) error {
	if doLargeSheet {
		styleID, err := excelInMemoryStructure.NewStyle(`{"font":{"color":"#EE2277"}}`)
		if err != nil {
			return err
		}
		if err := efficientExcelAPIWriter.SetRow("A1", []interface{}{
			excelize.Cell{StyleID: styleID, Value: "Data, > 10K lines (efficient)"}}); err != nil {
			return err
		}
	} else {
		styleID, err := excelInMemoryStructure.NewStyle(`{"font":{"color":"#EE2277"}}`)
		if err != nil {
			return err
		}
		if err = excelInMemoryStructure.SetCellStyle(sheet1, "A1", "C3", styleID); err != nil {
			return &Error{err: fmt.Errorf("SetCellStyle 1 %w", err)}
		}
		if err := excelInMemoryStructure.SetSheetRow(sheet1, "A1", &[]interface{}{"Data, <=10K lines (API)"}); err != nil {
			return &Error{err: fmt.Errorf("SetSheetRow 1 %w", err)}
		}
	}

	return nil
}

func ApplySmallSheetCellStyle(excelInMemoryStructure *excelize.File, startRow, maxCol, outputRow int, sheet1 string, styleID14 int) error {
	// set font style for range of cells written

	cellTopLeft, err := excelize.CoordinatesToCellName(1, startRow)
	if err != nil {
		return &Error{err: fmt.Errorf("CoordinatesToCellName 1 %w", err)}
	}

	cellBottomRight, err := excelize.CoordinatesToCellName(maxCol, outputRow)
	if err != nil {
		return &Error{err: fmt.Errorf("CoordinatesToCellName 2 %w", err)}
	}

	if err = excelInMemoryStructure.SetCellStyle(sheet1, cellTopLeft, cellBottomRight, styleID14); err != nil {
		return &Error{err: fmt.Errorf("SetCellStyle 2 %w", err)}
	}

	for i := startRow; i < outputRow; i++ { //!!! this may not be needed ?
		if err = excelInMemoryStructure.SetRowHeight(sheet1, i, 14); err != nil {
			return &Error{err: fmt.Errorf("SetRowHeight %w", err)}
		}
	}

	return nil
}
