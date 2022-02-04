package handler

import (
	"bufio"
	"context"
	"encoding/hex"

	"fmt"
	"io"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/headers"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"

	kafka "github.com/ONSdigital/dp-kafka/v3"

	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/xuri/excelize/v2"
)

const (
	maxAllowedRowCount       = 999900
	smallEnoughForFullFormat = 10000 // Not too large to achieve full formatting in memory
	maxSettableColumnWidths  = 500   // The maximum number of columns whose widths will be determined and set in Excel files whose source csv file has <= 'smallEnoughForFullFormat' lines. Apparently the max in the real dataset is 400, so we have a larger number just in case.
	columNotSet              = -1    // Magic number indicating column width has no determined value
	maxExcelizeColumnWidth   = 255   // Max column width that the excelize library will work with
)

// XlsxCreate is the handle for the CsvHandler event
type XlsxCreate struct {
	cfg         config.Config
	datasets    DatasetAPIClient
	s3Private   S3Client
	s3Public    S3Client
	vaultClient VaultClient
	producer    kafka.IProducer
	generator   Generator
}

// NewXlsxCreate a new CsvHandler
func NewXlsxCreate(cfg config.Config, d DatasetAPIClient, sPrivate S3Client, sPublic S3Client,
	v VaultClient, p kafka.IProducer, g Generator) *XlsxCreate {
	return &XlsxCreate{
		cfg:         cfg,
		datasets:    d,
		s3Private:   sPrivate,
		s3Public:    sPublic,
		vaultClient: v,
		producer:    p,
		generator:   g,
	}
}

// Handle takes a single event.
func (h *XlsxCreate) Handle(ctx context.Context, workerID int, msg kafka.Message) error {
	_ = workerID // to shut linter up
	kafkaEvent := &event.CantabularCsvCreated{}
	s := schema.CantabularCsvCreated

	if err := s.Unmarshal(msg.GetData(), kafkaEvent); err != nil {
		return &Error{
			err: fmt.Errorf("failed to unmarshal event: %w", err),
			logData: map[string]interface{}{
				"msg_data": msg.GetData(),
			},
		}
	}

	logData := log.Data{"event": kafkaEvent}
	log.Info(ctx, "event received", logData)

	if err := validateEvent(kafkaEvent); err != nil {
		return &Error{err: err,
			logData: logData,
		}
	}

	isPublished, err := h.isInstancePublished(ctx, kafkaEvent.InstanceID)
	if err != nil {
		return &Error{err: err,
			logData: logData,
		}
	}

	doLargeSheet := true

	if kafkaEvent.RowCount <= smallEnoughForFullFormat {
		// The number of lines in the CSV file is small enough to use the excelize API calls to create
		// an Excel file where we can determine and set the column widths to provide the user with
		// as good a presented Excel file as we can.
		// (As of Nov 2021 the excelize streaming library code does not allow setting of column widths,
		//  but we use the stream API calls for its speed and memory allocation efficiency for larger
		//  CSV files).
		// NOTE: the excelize libraries use of the word 'stream' is misleading as it's actually the
		// excelize libraries efficient mechanism of storing large sheets in memory and has nothing
		// to do with the meaning of the word 'streaming'.
		doLargeSheet = false
	}

	// Whilst running hundreds of full integration tests and observing memory usage on local macbook
	// by using 'docker stats' it was found that doing the following garbage collector call
	// helped imensely in keeping the HEAP memory down, thus putting the available memory for
	// the next run of this code in a better place to cope with a large file.
	defer runtime.GC()

	s3Path, err := h.processEventIntoXlsxFileOnS3(ctx, kafkaEvent, doLargeSheet, isPublished)
	if err != nil {
		return &Error{err: fmt.Errorf("faile in processEventIntoExcelFileOnS3"),
			logData: logData,
		}
	}

	numBytes, err := h.GetS3ContentLength(kafkaEvent, isPublished)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("failed to get S3 content length: %w", err),
			logData: logData,
		}
	}

	// Update instance with link to file
	if err := h.UpdateInstance(ctx, kafkaEvent, numBytes, isPublished, s3Path); err != nil {
		return errors.Wrapf(err, "failed to update instance")
	}

	return nil
}

// StreamAndWrite decrypt and stream the request file writing the content to the provided io.Writer.
func (h *XlsxCreate) StreamAndWrite(ctx context.Context, filenameCsv string, event *event.CantabularCsvCreated, w io.Writer, isPublished bool) (length int64, err error) {
	var s3ReadCloser io.ReadCloser
	var lengthPtr *int64

	if isPublished {
		s3ReadCloser, lengthPtr, err = h.s3Public.Get(filenameCsv)
		if err != nil {
			return 0, errors.Wrapf(err, "failed in Published Get")
		}
	} else {
		if h.cfg.EncryptionDisabled {
			s3ReadCloser, lengthPtr, err = h.s3Private.Get(filenameCsv)
			if err != nil {
				return 0, errors.Wrapf(err, "failed in Get")
			}
		} else {
			psk, err := h.getVaultKeyForCSVFile(event)
			if err != nil {
				return 0, errors.Wrapf(err, "failed in getVaultKeyForCSVFile")
			}

			s3ReadCloser, lengthPtr, err = h.s3Private.GetWithPSK(filenameCsv, psk)
			if err != nil {
				return 0, errors.Wrapf(err, "failed in GetWithPSK")
			}
		}
	}

	if lengthPtr != nil {
		length = *lengthPtr
	}

	defer closeAndLogError(ctx, s3ReadCloser)

	_, err = io.Copy(w, s3ReadCloser)
	if err != nil {
		return 0, errors.Wrapf(err, "failed in io.Copy")
	}

	return length, nil
}

func closeAndLogError(ctx context.Context, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Error(ctx, "error closing io.Closer", err)
	}
}

//!!! unit test this
func (h *XlsxCreate) getVaultKeyForCSVFile(event *event.CantabularCsvCreated) ([]byte, error) {
	vaultPath := fmt.Sprintf("%s/%s-%s-%s.csv", h.cfg.VaultPath, event.DatasetID, event.Edition, event.Version)

	pskStr, err := h.vaultClient.ReadKey(vaultPath, "key")
	if err != nil {
		return nil, errors.Wrapf(err, "for 'vaultPath': %s, failed in ReadKey", vaultPath)
	}

	psk, err := hex.DecodeString(pskStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed in DecodeString")
	}

	return psk, nil
}

//!!! unit test this
func validateEvent(kafkaEvent *event.CantabularCsvCreated) error {
	if kafkaEvent.RowCount > maxAllowedRowCount {
		return fmt.Errorf("full download too large to export to .xlsx file")
	}

	if kafkaEvent.InstanceID == "" {
		return fmt.Errorf("instanceID is empty")
	}

	return nil
}

//!!! unit test this
func (h *XlsxCreate) isInstancePublished(ctx context.Context, instanceID string) (bool, error) {
	instance, _, err := h.datasets.GetInstance(ctx, "", h.cfg.ServiceAuthToken, "", instanceID, headers.IfMatchAnyETag)
	if err != nil {
		return true, fmt.Errorf("failed to get instance: %w", err)
	}

	log.Info(ctx, "instance obtained from dataset API", log.Data{
		"instance_id":    instance.ID,
		"instance_state": instance.State,
	})

	isPublished := instance.State == dataset.StatePublished.String()

	return isPublished, nil
}

// processEventIntoXlsxFileOnS3 runs through the steps to stream in a csv file into an in memory representation of an xlsx file.
// This is then streamed out to an xlsx file on S3.
func (h *XlsxCreate) processEventIntoXlsxFileOnS3(ctx context.Context, kafkaEvent *event.CantabularCsvCreated, doLargeSheet bool, isPublished bool) (string, error) {
	// start creating the Excel file in its "in memory structure"
	excelInMemoryStructure := excelize.NewFile()
	sheet1 := "Sheet1"
	var efficientExcelAPIWriter *excelize.StreamWriter
	var err error

	if doLargeSheet {
		efficientExcelAPIWriter, err = excelInMemoryStructure.NewStreamWriter(sheet1) // have to start with the one and only default 'Sheet1'
		if err != nil {
			return "", fmt.Errorf("excel stream writer creation problem")
		}
	}

	excelInMemoryStructure.SetDefaultFont("Aerial")

	// Write header on first sheet, just to demonstrate ... this may not be needed - TBD
	if err = ApplyMainSheetHeader(excelInMemoryStructure, doLargeSheet, efficientExcelAPIWriter, sheet1); err != nil {
		if err != nil {
			return "", fmt.Errorf("ApplyMainSheetHeader failed: %w", err)
		}
	}

	if err = h.GetCSVtoExcelStructure(ctx, excelInMemoryStructure, kafkaEvent, doLargeSheet, efficientExcelAPIWriter, sheet1, isPublished); err != nil {
		if err != nil {
			return "", fmt.Errorf("GetCSVtoExcelStructure failed: %w", err)
		}
	}

	if err = h.AddMetaDataToExcelStructure(ctx, excelInMemoryStructure, kafkaEvent); err != nil {
		return "", fmt.Errorf("AddMetaDataToExcelStructure failed: %w", err)
	}

	// Rename the main sheet to 'Dataset'
	sheetDataset := "Dataset"
	excelInMemoryStructure.SetSheetName(sheet1, sheetDataset)

	// Set active sheet of the workbook.
	excelInMemoryStructure.SetActiveSheet(excelInMemoryStructure.GetSheetIndex(sheetDataset))

	s3Path, err := h.SaveExcelStructureToExcelFile(ctx, excelInMemoryStructure, kafkaEvent, isPublished)
	if err != nil {
		return "", fmt.Errorf("SaveExcelStructureToExcelFile failed: %w", err)
	}

	return s3Path, nil
}

// GetCSVtoExcelStructure streams in a line at a time from csv file from S3 bucket and
// inserts it into the Excel "in memory structure"
func (h *XlsxCreate) GetCSVtoExcelStructure(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated, doLargeSheet bool, efficientExcelAPIWriter *excelize.StreamWriter, sheet1 string, isPublished bool) error {
	var bucketName string
	var columnWidths [maxSettableColumnWidths]int

	if !doLargeSheet {
		// mark all column widths as unknown
		for i := 0; i < maxSettableColumnWidths; i++ {
			columnWidths[i] = columNotSet
		}
	}

	if isPublished {
		bucketName = h.s3Public.BucketName()
	} else {
		bucketName = h.s3Private.BucketName()
	}

	filenameCsv := generateS3FilenameCSV(event)

	// Create an io.Pipe to have the ability to read what is written to a writer
	csvReader, csvWriter := io.Pipe()

	downloadCtx, cancelDownload := context.WithCancel(ctx)
	defer cancelDownload()

	wgDownload := sync.WaitGroup{}
	wgDownload.Add(1)
	go func(ctx context.Context) {
		defer wgDownload.Done()

		numberOfBytesRead, err := h.StreamAndWrite(ctx, filenameCsv, event, csvWriter, isPublished)

		if err != nil {
			report := &Error{err: fmt.Errorf("StreamAndWrite failed, %w", err),
				logData: log.Data{"err": err, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}

			if closeErr := csvWriter.CloseWithError(report); closeErr != nil {
				log.Error(ctx, "error closing StreamAndWrite writerWithError", closeErr)
			}
		} else {
			log.Info(ctx, fmt.Sprintf(".csv file: %s, downloaded from bucket: %s, length: %d bytes", filenameCsv, bucketName, numberOfBytesRead))

			if closeErr := csvWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing StreamAndWrite writer", closeErr)
			}
		}
	}(downloadCtx)

	var startRow = 3
	var outputRow = startRow // this value chosen for test to visually see effect in Excel spreadsheet - this will probably need adjusting - TBD
	// AND most importantly to NOT touch any cells previously created with the excelize streamWriter mechanism

	styleID14, err := excelInMemoryStructure.NewStyle(`{"font":{"size":14}}`)
	if err != nil {
		return fmt.Errorf("NewStyle size 14 %w", err)
	}
	var incomingCsvRow = 0
	scanner := bufio.NewScanner(csvReader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			log.Info(ctx, "parent context closed - closing csv scanner loop ")
			if closeErr := csvWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing StreamAndWrite writer during context done signal", closeErr)
			}
			wgDownload.Wait()
			return fmt.Errorf("parent context closed in GetCSVtoExcelStructure")
		default:
			break
		}

		incomingCsvRow++
		line := scanner.Text()

		// split 'line' and do the Excel write at 'row' & deal with any errors
		columns := strings.Split(line, ",")
		nofColumns := len(columns)
		if nofColumns == 0 {
			return &Error{err: fmt.Errorf("downloaded .csv file has no columns at row %d", incomingCsvRow),
				logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}

		// Create row items from csv line
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
				colStr := ""
				if err == nil {
					rowItemsWithStyle = append(rowItemsWithStyle, valueFloat)
					colStr = fmt.Sprintf("%f", valueFloat)
				} else {
					rowItemsWithStyle = append(rowItemsWithStyle, value)
					colStr = value
				}
				if colID < maxSettableColumnWidths {
					l := len(colStr)
					if l > columnWidths[colID] {
						// update record of maximum column width
						columnWidths[colID] = l
					}
				}
			}
		}

		// Place row items into excelize data structure
		if doLargeSheet {
			cell, err := excelize.CoordinatesToCellName(1, outputRow)
			if err != nil {
				return &Error{err: err,
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
			if err := efficientExcelAPIWriter.SetRow(cell, rowItemsWithStyle); err != nil {
				return &Error{err: err,
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
		} else {
			addr, err := excelize.JoinCellName("A", outputRow)
			if err != nil {
				return &Error{err: fmt.Errorf("JoinCellName %w", err),
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
			if err := excelInMemoryStructure.SetSheetRow(sheet1, addr, &rowItemsWithStyle); err != nil {
				return &Error{err: fmt.Errorf("SetSheetRow 2 %w", err),
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
		}
		outputRow++
	}
	if err := scanner.Err(); err != nil {
		return &Error{err: fmt.Errorf("error whilst getting CSV row %w", err),
			logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
		}
	}

	// The whole CSV file has now been downloaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgDownload.Wait()

	if doLargeSheet {
		// Must now finish up the CSV excelize streamWriter calls before doing excelize API calls in building up metadata sheet:
		if err := efficientExcelAPIWriter.Flush(); err != nil {
			return &Error{err: err,
				logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}
	} else {
		// Process and apply column widths
		for i := 0; i < maxSettableColumnWidths; i++ {
			if columnWidths[i] != columNotSet {
				width := columnWidths[i] + 1 // add 1 to achieve slight visual space between columns and/or the vertical column lines
				if width > maxExcelizeColumnWidth {
					width = maxExcelizeColumnWidth
				}
				columnName, err := excelize.ColumnNumberToName(i + 1) // add 1, as column numbers start at 1 in excelize library
				if err != nil {
					return fmt.Errorf("ColumnNumberToName %w", err)
				}
				err = excelInMemoryStructure.SetColWidth(sheet1, columnName, columnName, float64(width))
				if err != nil {
					return fmt.Errorf("SetColWidth failed for Dataset sheet: %w", err)
				}
			}
		}
	}

	return nil
}

// SaveExcelStructureToExcelFile uses the excelize library Write function to effectively write out the Excel
// "in memory structure" to a stream that is then streamed directly into a file in S3 bucket.
// returns s3Location (path) or Error
func (h *XlsxCreate) SaveExcelStructureToExcelFile(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated, isPublished bool) (string, error) {
	var bucketName string
	if isPublished {
		bucketName = h.s3Public.BucketName()
	} else {
		bucketName = h.s3Private.BucketName()
	}

	filenameXlsx := generateS3FilenameXLSX(event)
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
			log.Info(ctx, fmt.Sprintf("finished writing file: %s, to pipe for bucket: %s", filenameXlsx, bucketName))

			if closeErr := xlsxWriter.Close(); closeErr != nil {
				log.Error(ctx, "error closing upload writer", closeErr)
			}
		}
	}()

	// Use the Upload function to read from the io.Pipe() Writer:
	s3Path, err := h.UploadXLSXFile(ctx, event, xlsxReader, isPublished, bucketName, filenameXlsx)
	if err != nil {
		if closeErr := xlsxWriter.Close(); closeErr != nil {
			log.Error(ctx, "error closing upload writer", closeErr)
		}

		return "", &Error{err: fmt.Errorf("failed to upload .xlsx file to S3 bucket: %w", err),
			logData: log.Data{
				"bucket":      bucketName,
				"instance_id": event.InstanceID,
			},
		}
	}

	// The whole XLSX file has now been uploaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgUpload.Wait()

	return s3Path, nil
}

// UploadXLSXFile uploads the provided file content to AWS S3
// returns s3Location (path) or Error
func (h *XlsxCreate) UploadXLSXFile(ctx context.Context, event *event.CantabularCsvCreated, file io.Reader, isPublished bool, bucketName string, filename string) (string, error) {
	if event.InstanceID == "" {
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

		// We use UploadWithContext because when processing an Excel file that is
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

			vaultPath := fmt.Sprintf("%s/%s-%s-%s.xlsx", h.cfg.VaultPath, event.DatasetID, event.Edition, event.Version)
			vaultKey := "key"

			log.Info(ctx, "writing key to vault", log.Data{"vault_path": vaultPath})

			if err := h.vaultClient.WriteKey(vaultPath, vaultKey, hex.EncodeToString(psk)); err != nil {
				return "", NewError(fmt.Errorf("WriteKey failed to write key to vault: %w", err),
					logData,
				)
			}

			// This code needs to use 'UploadWithPSKAndContext', because when processing an Excel file that is
			// nearly 1 million lines it has been seen to take over 45 seconds and if nomad has instructed a service
			// to shut down gracefully before installing a new version of this app, then without using context this
			// could cause problems.
			result, err := h.s3Private.UploadWithPSKAndContext(ctx, &s3manager.UploadInput{
				Body:   file,
				Bucket: &bucketName,
				Key:    &filename,
			}, psk)
			if err != nil {
				return "", NewError(fmt.Errorf("UploadWithPSKAndContext failed to upload encrypted file to S3: %w", err),
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

//!!! unit test this
// GetS3ContentLength obtains an S3 file size (in number of bytes) by calling Head Object
func (h *XlsxCreate) GetS3ContentLength(event *event.CantabularCsvCreated, isPublished bool) (int, error) {
	filename := generateS3FilenameXLSX(event)
	if isPublished {
		headOutput, err := h.s3Public.Head(filename)
		if err != nil {
			return 0, fmt.Errorf("public s3 head object error: %w", err)
		}
		return int(*headOutput.ContentLength), nil
	}
	headOutput, err := h.s3Private.Head(filename)
	if err != nil {
		return 0, fmt.Errorf("private s3 head object error: %w", err)
	}
	return int(*headOutput.ContentLength), nil
}

//!!! unit test this
// UpdateInstance updates the instance download CSV link using dataset API PUT /instances/{id} endpoint
// if the instance is published, then the s3Url will be set as public link and the instance state will be set to published
// otherwise, a private url will be generated and the state will not be changed
func (h *XlsxCreate) UpdateInstance(ctx context.Context, event *event.CantabularCsvCreated, size int, isPublished bool, s3Url string) error {
	xlsxDownload := &dataset.Download{
		Size: strconv.Itoa(size),
		URL: fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.xlsx",
			h.cfg.DownloadServiceURL,
			event.DatasetID,
			event.Edition,
			event.Version,
		),
	}

	if isPublished {
		xlsxDownload.Public = s3Url
	} else {
		xlsxDownload.Private = s3Url
	}

	log.Info(ctx, "updating dataset api with download link", log.Data{
		"isPublished":  isPublished,
		"xlsxDownload": xlsxDownload,
	})

	versionUpdate := dataset.Version{
		Downloads: map[string]dataset.Download{
			"XLS": *xlsxDownload,
		},
	}

	err := h.datasets.PutVersion(
		ctx,
		"",
		h.cfg.ServiceAuthToken,
		"",
		event.DatasetID,
		event.Edition,
		event.Version,
		versionUpdate)
	if err != nil {
		return fmt.Errorf("error while attempting update version downloads: %w", err)
	}

	return nil
}

// generateS3FilenameCSV generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID CSV file that is going to be read
func generateS3FilenameCSV(event *event.CantabularCsvCreated) string {
	return fmt.Sprintf("datasets/%s-%s-%s.csv", event.DatasetID, event.Edition, event.Version)

	// return fmt.Sprintf("instances/1000Kx50.csv")// OBSERVED: for non stream code this crashes using 13GB RAM in docker
	// return fmt.Sprintf("instances/50Kx50.csv") // OBSERVED this uses 1.7GB for non-large excel code
	// return fmt.Sprintf("instances/10Kx7.csv")
	// return fmt.Sprintf("instances/25Kx7.csv")
	// return fmt.Sprintf("instances/50Kx7.csv")
	// return fmt.Sprintf("instances/100Kx7.csv")
	// return fmt.Sprintf("instances/1000Kx7.csv")
}

// generateS3FilenameXLSX generates the S3 key (filename including `subpaths` after the bucket)
// for the provided instanceID XLSX file that is going to be written
func generateS3FilenameXLSX(event *event.CantabularCsvCreated) string {
	return fmt.Sprintf("datasets/%s-%s-%s.xlsx", event.DatasetID, event.Edition, event.Version)
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
			return fmt.Errorf("SetCellStyle 1 %w", err)
		}
		if err := excelInMemoryStructure.SetSheetRow(sheet1, "A1", &[]interface{}{"Data, <=10K lines (API)"}); err != nil {
			return fmt.Errorf("SetSheetRow 1 %w", err)
		}
	}

	return nil
}
