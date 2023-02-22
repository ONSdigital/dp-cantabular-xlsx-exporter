package handler

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
)

const (
	flexible     = "flexible"
	multivariate = "multivariate"
)

func (h *XlsxCreate) GetPlaceholderMetadata() *dataset.Metadata {
	return &dataset.Metadata{
		Version: dataset.Version{
			ReleaseDate: "2006-01-02T15:04:05.000Z",
		},
		DatasetDetails: dataset.DatasetDetails{
			Title: "Custom Dataset",
		},
	}
}

func (h *XlsxCreate) GetFilterDimensions(ctx context.Context, filterOutput filter.Model) ([]dataset.VersionDimension, error) {
	var areaType string
	for _, d := range filterOutput.Dimensions {
		if d.IsAreaType != nil && *d.IsAreaType {
			areaType = d.ID
		}
	}

	cReq := cantabular.GetDimensionsByNameRequest{
		Dataset: filterOutput.PopulationType,
	}
	for _, d := range filterOutput.Dimensions {
		cReq.DimensionNames = append(cReq.DimensionNames, d.ID)
	}
	resp, err := h.ctblr.GetDimensionsByName(ctx, cReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query dimensions")
	}

	var dims []dataset.VersionDimension
	for _, e := range resp.Dataset.Variables.Edges {
		isAreaType := e.Node.Name == areaType
		dim := dataset.VersionDimension{
			Label:       e.Node.Label,
			Description: e.Node.Description,
			IsAreaType:  &isAreaType,
		}
		dims = append(dims, dim)
	}

	return dims, nil
}

// AddMetaDataToExcelStructure reads in the metadata structure and extracts the desired items in the desired order
// and places them into the metadata sheet of the in-memory excelize library structure
func (h *XlsxCreate) AddMetaDataToExcelStructure(ctx context.Context, excelInMemoryStructure *excelize.File, event *event.CantabularCsvCreated) error {
	var err error

	logData := log.Data{
		"event": event,
	}

	req := dataset.GetVersionMetadataSelectionInput{
		UserAuthToken:    "",
		ServiceAuthToken: h.cfg.ServiceAuthToken,
		CollectionID:     "",
		DatasetID:        event.DatasetID,
		Edition:          event.Edition,
		Version:          event.Version,
		Dimensions:       event.Dimensions,
	}

	isFilterJob := event.FilterOutputID != ""
	var meta *dataset.Metadata
	var filterOutput filter.Model

	if isFilterJob {
		filterOutput, err = h.filterClient.GetOutput(ctx, "", h.cfg.ServiceAuthToken, "", "", event.FilterOutputID)
		if err != nil {
			return errors.Wrap(err, "failed to get filter output")
		}
	}
	if filterOutput.Type == multivariate {
		meta = h.GetPlaceholderMetadata()
	} else {
		meta, err = h.datasets.GetVersionMetadataSelection(ctx, req)
		if err != nil {
			return &Error{
				err:     errors.Wrap(err, "failed to get version metadata"),
				logData: logData,
			}
		}
	}
	metaExcel := "Metadata"

	// we don't need to save the newly created sheet number, because the caller of this function will set the active page to a different sheet
	_ = excelInMemoryStructure.NewSheet(metaExcel)

	rowNumber := 1

	columnAwidth := 1
	columnBwidth := 1

	var procErr error

	var sdcStatement = "Sometimes we need to make changes to data if it is possible to identify individuals. This is known as statistical disclosure control. In Census 2021, we: swapped records (targeted record swapping), for example, if a household was likely to be identified in datasets because it has unusual characteristics,"
	var sdcStatementRowTwo = "we swapped the record with a similar one from a nearby small area (very unusual households could be swapped with one in a nearby local authority) added small changes to some counts (cell key perturbation), for example, we might change a count of four to a three or a five – this might make small differences"
	var sdcStatementRowThree = "between tables depending on how the data are broken down when we applied perturbation"
	var areaTypeStatic = "Census 2021 statistics are published for a number of different geographies. These can be large, for example the whole of England, or small, for example an output area (OA), the lowest level of geography for which statistics are produced."
	var areaTypeStaticRowTwo = "For higher levels of geography, more detailed statistics can be produced. When a lower level of geography is used, such as output areas (which have a minimum of 100 persons), the statistics produced have less detail. This is to protect the confidentiality of people and ensure that individuals or"
	var areaTypeStaticRowThree = "their characteristics cannot be identified."
	var coverageStatic = "Census 2021 statistics are published for the whole of England and Wales. However, you can choose to filter areas by: country - (for example, Wales), region - (for example, London), local authority - (for example, Cornwall), health area – (for example, Clinical Commissioning Group),"
	var coverageStaticRowTwo = "statistical area - (for example, MSOA or LSOA)"
	var formatToParse = "2006-01-02T15:04:05.000Z"

	// place items in columns A and B, determine max column widths, and advance to next row
	processMetaElement := func(col1, col2 string, skipIfCol2Empty bool) {
		if procErr != nil {
			return
		}

		if skipIfCol2Empty && col2 == "" {
			return
		}

		addr, err := excelize.JoinCellName("A", rowNumber)
		if err != nil {
			procErr = errors.Wrap(err, "JoinCellName A")
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col1); err != nil {
			procErr = errors.Wrap(err, "SetCellValue A")
			return
		}
		addr, err = excelize.JoinCellName("B", rowNumber)
		if err != nil {
			procErr = errors.Wrap(err, "JoinCellName B")
			return
		}
		if err := excelInMemoryStructure.SetCellValue(metaExcel, addr, col2); err != nil {
			procErr = errors.Wrap(err, "SetCellValue B")
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
	date, err := time.Parse(formatToParse, meta.Version.ReleaseDate)
	if err != nil {
		return errors.Wrap(err, "unable to parse time")
	}
	processMetaElement("Release Date", date.Format(time.RFC822), true)
	processMetaElement("Dataset URL", meta.DatasetDetails.URI, true)
	processMetaElement("Unit of Measure", meta.DatasetDetails.UnitOfMeasure, true)

	if meta.DatasetDetails.Contacts != nil {
		rowNumber++
		processMetaElement("Contacts", "", false)
		for _, contacts := range *meta.DatasetDetails.Contacts {
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

	re := regexp.MustCompile("https?://([^/]+)")
	processMetaElement("Dataset Version URL", re.ReplaceAllString(meta.DatasetLinks.LatestVersion.URL, h.cfg.ExternalPrefixURL), true)
	processMetaElement("Statistical Disclosure Control Statement", sdcStatement, true)
	processMetaElement("", sdcStatementRowTwo, true)
	processMetaElement("", sdcStatementRowThree, true)
	processMetaElement("Area Type", areaTypeStatic, true)
	processMetaElement("", areaTypeStaticRowTwo, true)
	processMetaElement("", areaTypeStaticRowThree, true)

	var dims []dataset.VersionDimension

	if isFilterJob {
		dims, err = h.GetFilterDimensions(ctx, filterOutput)
		if err != nil {
			return &Error{
				err:     errors.Wrap(err, "failed to get filter dimensions"),
				logData: logData,
			}
		}
	} else {
		dims = meta.Version.Dimensions
	}

	for _, dim := range dims {
		if dim.IsAreaType != nil && *dim.IsAreaType {
			processMetaElement("Area Type Name", dim.Label, true)
			processMetaElement("Area Type Description", dim.Description, true)
		} else {
			processMetaElement("Variable Name", dim.Label, true)
		}
		processMetaElement("Quality Statement", dim.QualityStatementText, true)
		processMetaElement("Quality Statement URL", dim.QualityStatementURL, true)
	}

	processMetaElement("Coverage", coverageStatic, true)
	processMetaElement("", coverageStaticRowTwo, true)

	if !isFilterJob {
		versions, err := h.datasets.GetVersions(ctx, req.UserAuthToken, req.ServiceAuthToken, "", req.CollectionID, req.DatasetID, req.Edition, &dataset.QueryParams{Offset: 0, Limit: 100})
		if err != nil {
			return &Error{
				err:     errors.Wrap(err, "failed to get versions"),
				logData: logData,
			}
		}

		if len(versions.Items) > 0 {
			rowNumber++
			processMetaElement("Version History", "", false)
			for _, v := range versions.Items {
				rowNumber++
				processMetaElement("Version Number", strconv.Itoa(v.Version), true)
				date, err := time.Parse(formatToParse, v.ReleaseDate)
				if err != nil {
					return errors.Wrap(err, "unable to parse time")
				}
				processMetaElement("Release Date", date.Format(time.RFC822), true)

				if *v.Alerts != nil {
					for _, alerts := range *v.Alerts {
						processMetaElement("Reason for New Version", alerts.Description, true)
					}
				}
			}
		}

		if meta.DatasetDetails.RelatedContent != nil {
			for _, rc := range *meta.DatasetDetails.RelatedContent {
				rowNumber++
				processMetaElement("Related Content", "", false)
				rowNumber++
				processMetaElement("Title", rc.Title, true)
				processMetaElement("Description", rc.Description, true)
				processMetaElement("HRef", rc.HRef, true)

			}
		}
	}

	if procErr != nil {
		return errors.Wrap(procErr, "error in processing metadata")
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
