package handler

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// AddMetaDataToExcelStructure streams in a line at a time from .txt or .csvw file from S3 bucket and
// inserts it into the excel "in memory structure"
// !!! OR it might read the whole .txt or .csvw file into memory first (as its possibly less than 1 mega byte)
// and then inserts it into the excel "in memory structure"
func (h *CsvComplete) AddMetaDataToExcelStructure(excelFile *excelize.File) error {

	// !!! get the metadata

	//!!! add in the metadata to sheet 2, and deal with any errors
	// -=-=- : example test code for demo, using the excelize API calls ONLY (no more streaming) ...
	excelFile.NewSheet("Metadata")
	// Set value of a cell.
	excelFile.SetCellValue("Metadata", "A1", "Place")
	excelFile.SetCellValue("Metadata", "B1", "iiiiiiii")
	excelFile.SetCellValue("Metadata", "C1", "here ...")

	excelFile.SetCellValue("Metadata", "B9", "Hello")
	excelFile.SetCellValue("Metadata", "C10", "world")

	err := excelFile.SetColWidth("Metadata", "A", "B", 8) // !!! the max can be 255, so clamp in code at some point !!!
	// NOTE: the above sets the width to 8 times the max width of a character in a font and does not do proportionality for different width characters in a font
	// ... so, it would be best if we used a fixed width font !!! ... not Aerial as in the dp-dataset-exporter-xlsx
	//!!! also if the max number of characters in a column is say 10, then set the coumn width to 10+1 => 11
	if err != nil {
		return fmt.Errorf("SetColWidth failed: %w", err)
	}
	return nil
}
