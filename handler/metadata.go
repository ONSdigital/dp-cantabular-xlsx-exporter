package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
)

// AddMetaDataToExcelStructure reads in the metadata structure and extracts the desired items in the desired order
// and places them into the metadata sheet of the in-memory excelize library structure
func (h *XlsxCreate) AddMetaDataToExcelStructure(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated) error {
	logData := log.Data{
		"event": event,
	}

	req := dataset.GetVersionMetadataSelectionInput{
		DatasetID:  event.DatasetID,
		Edition:    event.Edition,
		Version:    event.Version,
		Dimensions: event.DimensionsID,
	}

	meta, err := h.datasets.GetVersionMetadataSelection(ctx, req)
	if err != nil {
		return &Error{
			err:     errors.Wrap(err, "failed to get version metadata"),
			logData: logData,
		}
	}

	metaExcel := "Metadata"

	// we don't need to save the newly created sheet number, because the caller of this function will set the active page to a different sheet
	_ = excelInMemoryStructure.NewSheet(metaExcel)

	rowNumber := 1

	columnAwidth := 1
	columnBwidth := 1

	processError := false

	var processErrorStr error

	// place items in columns A and B, determine max column widths, and advance to next row
	processMetaElement := func(col1, col2 string, skipIfCol2Empty bool) {
		if processError {
			return
		}

		if skipIfCol2Empty && col2 == "" {
			return
		}

		addr, err := excelize.JoinCellName("A", rowNumber)
		if err != nil {
			processError = true
			processErrorStr = errors.Wrap(err, "JoinCellName A")
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col1); err != nil {
			processError = true
			processErrorStr = errors.Wrap(err, "SetCellValue A")
			return
		}
		addr, err = excelize.JoinCellName("B", rowNumber)
		if err != nil {
			processError = true
			processErrorStr = errors.Wrap(err, "JoinCellName B")
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col2); err != nil {
			processError = true
			processErrorStr = errors.Wrap(err, "SetCellValue B")
			return
		}

		if len(col1) > columnAwidth {
			columnAwidth = len(col1)
		}

		if len(col2) > columnBwidth {
			columnBwidth = len(col2)
		}
		rowNumber++
	}

	// TODO the below fields are only an initial prototype, further adjustments will be needed we get the full metadata structure definition
	processMetaElement("Title", meta.DatasetDetails.Title, true)
	processMetaElement("Description", meta.DatasetDetails.Description, true)
	processMetaElement("Release Date", meta.Version.ReleaseDate, true)
	processMetaElement("Dataset URL", meta.DatasetDetails.URI, true)
	processMetaElement("Unit of Measure", meta.DatasetDetails.UnitOfMeasure, true)

	if meta.DatasetDetails.Contacts != nil {
		rowNumber++
		processMetaElement("Contacts", "", false)
		fmt.Printf("contacts struct: %v\n", *meta.DatasetDetails.Contacts)
		for _, contacts := range *meta.DatasetDetails.Contacts {
			if contacts.Name != "" {
				processMetaElement("", contacts.Name, true)
			}
			if contacts.Email != "" {
				processMetaElement("", contacts.Email, true)
			}
			if contacts.Telephone != "" {
				processMetaElement("", contacts.Telephone, true)
			}
			rowNumber++
		}
	}

	if meta.Version.Alerts != nil {
		processMetaElement("Alerts", "", false)
		for _, alerts := range *meta.Version.Alerts {
			if alerts.Date != "" {
				processMetaElement("", alerts.Date, true)
			}
			if alerts.Description != "" {
				processMetaElement("", alerts.Description, true)
			}
			if alerts.Type != "" {
				processMetaElement("", alerts.Type, true)
			}
			rowNumber++
		}
	}

	processMetaElement("Quality and methodology information", meta.DatasetDetails.QMI.URL, true)

	rowNumber++

	processMetaElement("Version", strconv.Itoa(meta.Version.Version), true)
	processMetaElement("Dataset version", meta.DatasetLinks.LatestVersion.URL, true)

	if processError {
		return errors.Wrap(processErrorStr, "error in processing metadata")
	}

	if columnAwidth > maxExcelizeColumnWidth {
		columnAwidth = maxExcelizeColumnWidth
	}

	if columnBwidth > maxExcelizeColumnWidth {
		columnBwidth = maxExcelizeColumnWidth
	}

	err = excelInMemoryStructure.SetColWidth("Metadata", "A", "A", float64(columnAwidth))
	if err != nil {
		return errors.Wrap(err, "SetColWidth A failed")
	}

	err = excelInMemoryStructure.SetColWidth("Metadata", "B", "B", float64(columnBwidth))
	if err != nil {
		return errors.Wrap(err, "SetColWidth B failed")
	}

	return nil
}
