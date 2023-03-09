package handler

import (
	"bufio"
	"context"
	"encoding/csv"
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
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/headers"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"

	kafka "github.com/ONSdigital/dp-kafka/v3"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/xuri/excelize/v2"

	"github.com/ONSdigital/log.go/v2/log"
)

const (
	MaxAllowedRowCount       = 999900
	smallEnoughForFullFormat = 10000 // Not too large to achieve full formatting in memory
	maxSettableColumnWidths  = 500   // The maximum number of columns whose widths will be determined and set in Excel files whose source csv file has <= 'smallEnoughForFullFormat' lines. Apparently the max in the real dataset is 400, so we have a larger number just in case.
	columNotSet              = -1    // Magic number indicating column width has no determined value
	maxExcelizeColumnWidth   = 255   // Max column width that the excelize library will work with
)

// XlsxCreate is the handle for the CsvHandler event
type XlsxCreate struct {
	cfg                   config.Config
	ctblr                 CantabularClient
	datasets              DatasetAPIClient
	s3Private             S3Client
	s3Public              S3Client
	vaultClient           VaultClient
	filterClient          FilterAPIClient
	populationTypesClient PopulationTypesAPIClient
	producer              kafka.IProducer
	generator             Generator
}

// NewXlsxCreate a new CsvHandler
func NewXlsxCreate(cfg config.Config, d DatasetAPIClient, sPrivate S3Client, sPublic S3Client,
	v VaultClient, f FilterAPIClient, g Generator, p PopulationTypesAPIClient, cc CantabularClient) *XlsxCreate {
	return &XlsxCreate{
		cfg:                   cfg,
		datasets:              d,
		s3Private:             sPrivate,
		s3Public:              sPublic,
		vaultClient:           v,
		filterClient:          f,
		generator:             g,
		populationTypesClient: p,
		ctblr:                 cc,
	}
}

// Handle takes a single event.
func (h *XlsxCreate) Handle(ctx context.Context, workerID int, msg kafka.Message) error {
	_ = workerID // to shut linter up
	kafkaEvent := &event.CantabularCsvCreated{}
	s := schema.CantabularCsvCreated

	if err := s.Unmarshal(msg.GetData(), kafkaEvent); err != nil {
		return &Error{
			err: errors.Wrap(err, "failed to unmarshal event"),
			logData: map[string]interface{}{
				"msg_data": msg.GetData(),
			},
		}
	}

	logData := log.Data{"event": kafkaEvent}
	log.Info(ctx, "event received", logData)

	if err := ValidateEvent(kafkaEvent); err != nil {
		return &Error{err: errors.Wrap(err, "failed to validate event"),
			logData: logData,
		}
	}

	var isPublished bool
	isFilterJob := kafkaEvent.FilterOutputID != ""
	if isFilterJob {
		var err error
		isPublished, err = h.isFilterPublished(ctx, kafkaEvent.FilterOutputID)
		if err != nil {
			return errors.Wrap(err, "failed to get filter info")
		}
	} else {
		var err error
		isPublished, err = h.IsInstancePublished(ctx, kafkaEvent.InstanceID)
		if err != nil {
			return &Error{err: errors.Wrap(err, "failed in IsInstancePublished"),
				logData: logData,
			}
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
	// helped immensely in keeping the HEAP memory down, thus putting the available memory for
	// the next run of this code in a better place to cope with a large file.
	defer runtime.GC()

	s3Path, fileName, err := h.processEventIntoXlsxFileOnS3(ctx, kafkaEvent, doLargeSheet, isPublished)
	if err != nil {
		return &Error{err: errors.Wrap(err, "failed in processEventIntoExcelFileOnS3"),
			logData: logData,
		}
	}

	numBytes, err := h.GetS3ContentLength(kafkaEvent, isPublished)
	if err != nil {
		return &Error{
			err:     errors.Wrap(err, "failed to get S3 content length"),
			logData: logData,
		}
	}

	if kafkaEvent.FilterOutputID != "" { // condition for filtered job
		if err := h.UpdateFilterOutput(ctx, kafkaEvent.FilterOutputID, numBytes, isPublished, s3Path, fileName); err != nil {
			return errors.Wrap(err, "failed to update filter output")
		}
	} else {
		// Update instance with link to file
		if err := h.UpdateInstance(ctx, kafkaEvent, numBytes, isPublished, s3Path, fileName); err != nil {
			return errors.Wrap(err, "failed to update instance")
		}
	}

	return nil
}

func ValidateEvent(kafkaEvent *event.CantabularCsvCreated) error {
	if kafkaEvent.RowCount > MaxAllowedRowCount {
		return ErrorStack("full download too large to export to .xlsx file")
	}

	if kafkaEvent.InstanceID == "" {
		return ErrorStack("instanceID is empty")
	}

	return nil
}

func (h *XlsxCreate) IsInstancePublished(ctx context.Context, instanceID string) (bool, error) {
	instance, _, err := h.datasets.GetInstance(ctx, "", h.cfg.ServiceAuthToken, "", instanceID, headers.IfMatchAnyETag)
	if err != nil {
		return true, errors.Wrap(err, "failed to get instance")
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
func (h *XlsxCreate) processEventIntoXlsxFileOnS3(ctx context.Context, kafkaEvent *event.CantabularCsvCreated, doLargeSheet bool, isPublished bool) (string, string, error) {
	// start creating the Excel file in its "in memory structure"
	excelInMemoryStructure := excelize.NewFile()
	sheet1 := "Sheet1"
	var efficientExcelAPIWriter *excelize.StreamWriter
	var err error

	if doLargeSheet {
		efficientExcelAPIWriter, err = excelInMemoryStructure.NewStreamWriter(sheet1) // have to start with the one and only default 'Sheet1'
		if err != nil {
			return "", "", ErrorStack("excel stream writer creation problem")
		}
	}

	excelInMemoryStructure.SetDefaultFont("Aerial")

	if err = h.GetCSVtoExcelStructure(ctx, excelInMemoryStructure, kafkaEvent, doLargeSheet, efficientExcelAPIWriter, sheet1, isPublished); err != nil {
		if err != nil {
			return "", "", errors.Wrap(err, "GetCSVtoExcelStructure failed")
		}
	}

	if err = h.AddMetaDataToExcelStructure(ctx, excelInMemoryStructure, kafkaEvent); err != nil {
		return "", "", errors.Wrap(err, "AddMetaDataToExcelStructure failed")
	}

	// Rename the main sheet to 'Dataset'
	sheetDataset := "Dataset"
	excelInMemoryStructure.SetSheetName(sheet1, sheetDataset)

	// Set active sheet of the workbook.
	excelInMemoryStructure.SetActiveSheet(excelInMemoryStructure.GetSheetIndex(sheetDataset))

	s3Path, fileName, err := h.SaveExcelStructureToExcelFile(ctx, excelInMemoryStructure, kafkaEvent, isPublished)
	if err != nil {
		return "", "", errors.Wrap(err, "SaveExcelStructureToExcelFile failed")
	}

	return s3Path, fileName, nil
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
			report := &Error{err: errors.Wrap(err, "StreamAndWrite failed"),
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

	var outputRow = 1

	styleID14, err := excelInMemoryStructure.NewStyle(`{"font":{"size":14}}`)
	if err != nil {
		return errors.Wrap(err, "NewStyle size 14")
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
			return errors.Wrap(err, "parent context closed in GetCSVtoExcelStructure")
		default:
			// This default & break is here so that we don't STOP in this select{} waiting on channel(s).
			// We only want to check for any ctx.Done() and take appropriate action, or with this 'default'
			// exit the 'select' to do work.
			// It is not 'ineffective' as the the IDE might be indicating and is here by design.
			break
		}

		incomingCsvRow++

		cr := csv.NewReader(strings.NewReader(scanner.Text()))
		columns, err := cr.Read()
		if err != nil {
			return &Error{err: errors.Wrapf(err, "error parsing csv row records/cells at row %d", incomingCsvRow),
				logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv},
			}
		}

		nofColumns := len(columns)
		if nofColumns == 0 {
			return &Error{err: errors.Wrapf(err, "downloaded .csv file has no columns at row %d", incomingCsvRow),
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
				return &Error{err: errors.Wrap(err, "JoinCellName"),
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
			if err := excelInMemoryStructure.SetSheetRow(sheet1, addr, &rowItemsWithStyle); err != nil {
				return &Error{err: errors.Wrap(err, "SetSheetRow 2"),
					logData: log.Data{"event": event, "bucketName": bucketName, "filenameCsv": filenameCsv, "incomingCsvRow": incomingCsvRow},
				}
			}
		}
		outputRow++
	}
	if err := scanner.Err(); err != nil {
		return &Error{err: errors.Wrap(err, "error whilst getting CSV row"),
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
					return errors.Wrap(err, "ColumnNumberToName")
				}
				err = excelInMemoryStructure.SetColWidth(sheet1, columnName, columnName, float64(width))
				if err != nil {
					return errors.Wrap(err, "SetColWidth failed for Dataset sheet")
				}
			}
		}
	}

	return nil
}

// SaveExcelStructureToExcelFile uses the excelize library Write function to effectively write out the Excel
// "in memory structure" to a stream that is then streamed directly into a file in S3 bucket.
// returns s3Location (path) or Error
func (h *XlsxCreate) SaveExcelStructureToExcelFile(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated, isPublished bool) (string, string, error) {
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

		return "", "", &Error{err: errors.Wrap(err, "failed to upload .xlsx file to S3 bucket"),
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

	return s3Path, filenameXlsx, nil
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
		"bucket":       bucketName,
		"filename":     filename,
		"is_published": isPublished,
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
			return "", NewError(
				errors.Wrap(err, "UploadWithContext failed to upload published file to S3"),
				logData,
			)
		}
		resultPath = result.Location
	} else {
		log.Info(ctx, "uploading encrypted file to S3", logData)

		psk, err := h.generator.NewPSK()
		if err != nil {
			return "", NewError(errors.Wrap(err, "NewPSK failed to generate a PSK for encryption"),
				logData,
			)
		}

		vaultPath := fmt.Sprintf("%s/%s-%s-%s.xlsx", h.cfg.VaultPath, event.DatasetID, event.Edition, event.Version)
		vaultKey := "key"

		log.Info(ctx, "writing key to vault", log.Data{"vault_path": vaultPath})

		if err := h.vaultClient.WriteKey(vaultPath, vaultKey, hex.EncodeToString(psk)); err != nil {
			return "", NewError(errors.Wrap(err, "WriteKey failed to write key to vault"),
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
			return "", NewError(errors.Wrap(err, "UploadWithPSKAndContext failed to upload encrypted file to S3"),
				logData,
			)
		}
		resultPath = result.Location
	}

	s3Location, err := url.PathUnescape(resultPath)
	if err != nil {
		logData["location"] = resultPath
		return "", NewError(errors.Wrap(err, "failed to unescape S3 path location"),
			logData,
		)
	}
	return s3Location, nil
}

// GetS3ContentLength obtains an S3 file size (in number of bytes) by calling Head Object
func (h *XlsxCreate) GetS3ContentLength(event *event.CantabularCsvCreated, isPublished bool) (int, error) {
	filename := generateS3FilenameXLSX(event)
	if isPublished {
		headOutput, err := h.s3Public.Head(filename)
		if err != nil {
			return 0, errors.Wrap(err, "public s3 head object error")
		}
		return int(*headOutput.ContentLength), nil
	}
	headOutput, err := h.s3Private.Head(filename)
	if err != nil {
		return 0, errors.Wrap(err, "private s3 head object error")
	}
	return int(*headOutput.ContentLength), nil
}

func (h *XlsxCreate) isFilterPublished(ctx context.Context, filterOutputID string) (bool, error) {
	model, err := h.filterClient.GetOutput(ctx, "", h.cfg.ServiceAuthToken, "", "", filterOutputID)
	if err != nil {
		return false, &Error{
			err:     errors.Wrap(err, "failed to get filter"),
			logData: log.Data{"filter_output_id": filterOutputID},
		}
	}

	return model.IsPublished, nil
}

func (h *XlsxCreate) UpdateFilterOutput(ctx context.Context, filterOutputID string, size int, isPublished bool, s3Url, filename string) error {
	log.Info(ctx, "Updating filter output with download link")

	download := filter.Download{
		URL:     fmt.Sprintf("%s/downloads/filter-outputs/%s.xlsx", h.cfg.DownloadServiceURL, filterOutputID),
		Size:    fmt.Sprintf("%d", size),
		Skipped: false,
	}

	if isPublished {
		download.Public = fmt.Sprintf("%s/%s",
			h.cfg.S3PublicURL,
			filename,
		)
	} else {
		download.Private = s3Url
	}

	m := filter.Model{
		Downloads: map[string]filter.Download{
			"XLS": download,
		},
		IsPublished: isPublished,
	}

	if err := h.filterClient.UpdateFilterOutput(ctx, "", h.cfg.ServiceAuthToken, "", filterOutputID, &m); err != nil {
		return errors.Wrap(err, "failed to update filter output")
	}

	return nil
}

// UpdateInstance updates the instance download CSV link using dataset API PUT /instances/{id} endpoint
// if the instance is published, then the s3Url will be set as public link and the instance state will be set to published
// otherwise, a private url will be generated and the state will not be changed
func (h *XlsxCreate) UpdateInstance(ctx context.Context, event *event.CantabularCsvCreated, size int, isPublished bool, s3Url, filename string) error {
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
		xlsxDownload.Public = fmt.Sprintf("%s/%s",
			h.cfg.S3PublicURL,
			filename,
		)
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
		versionUpdate,
	)
	if err != nil {
		return errors.Wrap(err, "error while attempting update version downloads")
	}

	return nil
}

func generateS3FilenameCSV(event *event.CantabularCsvCreated) string {
	return fmt.Sprintf("datasets/%s", event.FileName)
}

func generateS3FilenameXLSX(event *event.CantabularCsvCreated) string {
	return fmt.Sprintf("datasets/%s", strings.Replace(event.FileName, ".csv", ".xlsx", 1))
}
