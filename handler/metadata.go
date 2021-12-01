package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/xuri/excelize/v2"
)

// AddMetaDataToExcelStructure reads in the metadata structure and extracts the desired items in the desired order
// and places them into the metadata sheet of the in-memory excelize library structure
func (h *XlsxCreate) AddMetaDataToExcelStructure(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated) error {
	logData := log.Data{
		"event": event,
	}

	meta, err := h.datasets.GetVersionMetadata(ctx, "", h.cfg.ServiceAuthToken, "", event.DatasetID, event.Edition, event.Version)
	if err != nil {
		return &Error{
			err:     fmt.Errorf("failed to get version metadata: %w", err),
			logData: logData,
		}
	}

	log.Info(ctx, "metadata structure", log.Data{ // TODO this is for test, remove at some point
		"metadata_struct": meta,
	})

	metaExcel := "Metadata"

	// we don't need to save the newly created sheet number, because the caller of this function will set the active page to a different sheet
	_ = excelInMemoryStructure.NewSheet(metaExcel)

	rowNumber := 1

	columnAwidth := 1 //change to A, B
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
			processErrorStr = fmt.Errorf("JoinCellName A: %w", err)
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col1); err != nil {
			processError = true
			processErrorStr = fmt.Errorf("SetCellValue A: %w", err)
			return
		}
		addr, err = excelize.JoinCellName("B", rowNumber)
		if err != nil {
			processError = true
			processErrorStr = fmt.Errorf("JoinCellName B: %w", err)
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col2); err != nil {
			processError = true
			processErrorStr = fmt.Errorf("SetCellValue B: %w", err)
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

	processMetaElement("Qulaity and methodology information", meta.DatasetDetails.QMI.URL, true)

	rowNumber++

	processMetaElement("Version", strconv.Itoa(meta.Version.Version), true)

	if processError {
		return fmt.Errorf("error in processing metadata: %w", processErrorStr)
	}

	if columnAwidth > maxExcelizeColumnWidth {
		columnAwidth = maxExcelizeColumnWidth
	}

	if columnBwidth > maxExcelizeColumnWidth {
		columnBwidth = maxExcelizeColumnWidth
	}

	err = excelInMemoryStructure.SetColWidth("Metadata", "A", "A", float64(columnAwidth))
	if err != nil {
		return fmt.Errorf("SetColWidth A failed: %w", err)
	}

	err = excelInMemoryStructure.SetColWidth("Metadata", "B", "B", float64(columnBwidth))
	if err != nil {
		return fmt.Errorf("SetColWidth B failed: %w", err)
	}

	return nil
}
