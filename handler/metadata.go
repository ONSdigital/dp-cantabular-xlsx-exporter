package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/xuri/excelize/v2"
)

// AddMetaDataToExcelStructure streams in a line at a time from .txt or .csvw file from S3 bucket and
// inserts it into the excel "in memory structure"
// !!! OR it might read the whole .txt or .csvw file into memory first (as its possibly less than 1 mega byte)
// and then inserts it into the excel "in memory structure"
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

	// TODO place proper comments to describe what the below function does

	processMetaElement := func(col1, col2 string, skipIfCol2Empty bool) {
		if processError {
			return
		}
		fmt.Printf("%s: %s\n", col1, col2) // TODO remove this once development is complete

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

	processMetaElement("Title", meta.DatasetDetails.Title, true)
	processMetaElement("Description", meta.DatasetDetails.Description, true)
	processMetaElement("Release Date", meta.Version.ReleaseDate, true)
	processMetaElement("URL", meta.DatasetDetails.URI, true) // TODO which entry box in florence causes this to be populated, as this is printing a blank when I've entered something into the URI box?
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
	fmt.Printf("meta struct: %v\n", meta)                            // TODO trash after test
	spew.Dump(meta)                                                  // TODO trash after test
	log.Info(ctx, "full meta struct", log.Data{"meta struct": meta}) // TODO trash after test

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

	csvVal, csvOk := meta.Version.Downloads["csv"]
	csvwVal, csvwOk := meta.Version.Downloads["csvw"]
	txtVal, txtOk := meta.Version.Downloads["txt"]
	xlsxVal, xlsxOk := meta.Version.Downloads["xlsx"]

	if csvOk || csvwOk || txtOk || xlsxOk {

		processMetaElement("Downloads:", "", false)
		if csvOk && csvVal.URL != "" {
			processMetaElement("    csv:", csvVal.URL, true)
			// we also need to show the values in csvVal
		}
		if csvwOk && csvwVal.URL != "" {
			processMetaElement("    csvw:", csvwVal.URL, true)
			// we also need to show the values in csvwVal
		}
		if txtOk && txtVal.URL != "" {
			processMetaElement("    txt:", txtVal.URL, true)
			// we also need to show the values in txtVal
		}
		if xlsxOk && xlsxVal.URL != "" {
			processMetaElement("    xlsx:", xlsxVal.URL, true)
			// we also need to show the values in xlsxVal
		}
	}

	rowNumber++

	processMetaElement("Version", strconv.Itoa(meta.Version.Version), true)

	if processError {
		return fmt.Errorf("error in processing metadata: %w", processErrorStr)
	}

	if columnAwidth > 255 {
		columnAwidth = 255
	} // won't work properly because of indentation

	if columnBwidth > 255 {
		columnBwidth = 255
	}

	// fmt.Printf("column A width: %d\n", columnAwidth)

	err = excelInMemoryStructure.SetColWidth("Metadata", "A", "A", float64(columnAwidth)) // !!! the max can be 255, so clamp in code at some point !!!
	// NOTE: the above sets the widt to 8 times the max width of a character in a font and does not do proportionality for different width characters in a font
	// ... so, it would be best if we used a fixed width font !!! ... not Aerial as in the dp-dataset-exporter-xlsx
	//!!! also if the max number of characters in a column is say 10, then set the coumn width to 10+1 => 11
	if err != nil {
		fmt.Printf("SetColWidth A failed: %w", err)

		// return proper error
	}

	// fmt.Printf("column B width: %d\n", columnBwidth)

	err = excelInMemoryStructure.SetColWidth("Metadata", "B", "B", float64(columnBwidth)) // !!! the max can be 255, so clamp in code at some point !!!
	// NOTE: the above sets the widt to 8 times the max width of a character in a font and does not do proportionality for different width characters in a font
	// ... so, it would be best if we used a fixed width font !!! ... not Aerial as in the dp-dataset-exporter-xlsx
	//!!! also if the max number of characters in a column is say 10, then set the coumn width to 10+1 => 11
	if err != nil {
		fmt.Printf("SetColWidth B failed: %w", err)

		// return proper error
	}

	return nil
}
