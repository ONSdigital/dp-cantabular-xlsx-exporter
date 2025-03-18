package handler_test

import (
	"context"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler/mock"
	"github.com/xuri/excelize/v2"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	testFilterOutputIDMultivariate = "test-mv"
	testFilterOutputIDFlexible     = "test-fl"
	testFilterOutputIDCustom       = "test-cu"
	multivariateString             = "multivariate"
)

var (
	trueVal = true
)

func TestAddMetaDataToExcelStructure(t *testing.T) {
	ctx := context.Background()

	Convey("Given a handler with a healthy dataset, filter and cantabular client", t, func() {
		datasetAPIMock := mock.DatasetAPIClientMock{
			GetVersionMetadataSelectionFunc: func(_ context.Context, _ dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error) {
				return &dataset.Metadata{
					Version: dataset.Version{
						ReleaseDate: "2006-01-02T15:04:05.000Z",
					},
				}, nil
			},
			GetVersionsFunc: func(_ context.Context, _, _, _, _, _, _ string, _ *dataset.QueryParams) (dataset.VersionsList, error) {
				return dataset.VersionsList{}, nil
			},
		}
		filterAPIMock := mock.FilterAPIClientMock{
			GetOutputFunc: func(_ context.Context, _, _, _, _, id string) (filter.Model, error) {
				var resp filter.Model
				if id == testFilterOutputIDFlexible {
					resp.Type = "flexible"
				}
				if id == testFilterOutputIDMultivariate {
					resp.Type = multivariateString
				}
				if id == testFilterOutputIDCustom {
					resp.Type = multivariateString
					resp.Custom = &trueVal
				}
				return resp, nil
			},
		}
		ctblrClientMock := mock.CantabularClientMock{
			GetDimensionsByNameFunc: func(_ context.Context, _ cantabular.GetDimensionsByNameRequest) (*cantabular.GetDimensionsResponse, error) {
				return &cantabular.GetDimensionsResponse{}, nil
			},
		}
		h := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, &filterAPIMock, nil, nil, &ctblrClientMock)

		Convey("Given a valid execlize file", func() {
			xlsx := excelize.NewFile()

			Convey("When AddMetaDataToExcelStructure is called with an event without a filter output ID", func() {
				e := event.CantabularCsvCreated{
					InstanceID: testInstanceID,
					DatasetID:  testDatasetID,
					Edition:    testEdition,
					Version:    testVersion,
				}
				err := h.AddMetaDataToExcelStructure(ctx, xlsx, &e)
				So(err, ShouldBeNil)

				So(filterAPIMock.GetOutputCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.GetVersionMetadataSelectionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionsCalls(), ShouldHaveLength, 1)
				So(ctblrClientMock.GetDimensionsByNameCalls(), ShouldHaveLength, 0)
			})

			Convey("When AddMetaDataToExcelStructure is called with an event with a filter output ID for a flexible filter", func() {
				e := event.CantabularCsvCreated{
					InstanceID:     testInstanceID,
					DatasetID:      testDatasetID,
					Edition:        testEdition,
					Version:        testVersion,
					FilterOutputID: testFilterOutputIDFlexible,
				}
				err := h.AddMetaDataToExcelStructure(ctx, xlsx, &e)
				So(err, ShouldBeNil)

				So(filterAPIMock.GetOutputCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionMetadataSelectionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionsCalls(), ShouldHaveLength, 1)
				So(ctblrClientMock.GetDimensionsByNameCalls(), ShouldHaveLength, 1)
			})

			Convey("When AddMetaDataToExcelStructure is called with an event with a filter output ID for a multivariate filter", func() {
				e := event.CantabularCsvCreated{
					InstanceID:     testInstanceID,
					DatasetID:      testDatasetID,
					Edition:        testEdition,
					Version:        testVersion,
					FilterOutputID: testFilterOutputIDMultivariate,
				}
				err := h.AddMetaDataToExcelStructure(ctx, xlsx, &e)
				So(err, ShouldBeNil)

				So(filterAPIMock.GetOutputCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionMetadataSelectionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionsCalls(), ShouldHaveLength, 1)
				So(ctblrClientMock.GetDimensionsByNameCalls(), ShouldHaveLength, 1)
			})

			Convey("When AddMetaDataToExcelStructure is called with an event with a filter output ID for a custom filter", func() {
				e := event.CantabularCsvCreated{
					InstanceID:     testInstanceID,
					DatasetID:      testDatasetID,
					Edition:        testEdition,
					Version:        testVersion,
					FilterOutputID: testFilterOutputIDCustom,
				}
				err := h.AddMetaDataToExcelStructure(ctx, xlsx, &e)
				So(err, ShouldBeNil)

				So(filterAPIMock.GetOutputCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionMetadataSelectionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.GetVersionsCalls(), ShouldHaveLength, 0)
				So(ctblrClientMock.GetDimensionsByNameCalls(), ShouldHaveLength, 1)
			})
		})
	})
}

func TestGenerateCustomTitle(t *testing.T) {
	Convey("Given a handler", t, func() {
		h := &handler.XlsxCreate{}

		Convey("given a set of 3 filter dimensions", func() {
			dims := []filter.ModelDimension{
				{
					Label: "Label 1",
				},
				{
					Label: "Label 2",
				},
				{
					Label: "Label 3",
				},
			}

			Convey("when GenerateCustomTitle is called", func() {
				title := h.GenerateCustomTitle(dims)
				expected := "Label 1, Label 2 and Label 3"
				So(title, ShouldResemble, expected)
			})
		})

		Convey("given a set of 2 filter dimensions", func() {
			dims := []filter.ModelDimension{
				{
					Label: "Label 1",
				},
				{
					Label: "Label 2",
				},
			}

			Convey("when GenerateCustomTitle is called", func() {
				title := h.GenerateCustomTitle(dims)
				expected := "Label 1 and Label 2"
				So(title, ShouldResemble, expected)
			})
		})

		Convey("given a set of 1 filter dimensions", func() {
			dims := []filter.ModelDimension{
				{
					Label: "Label 1",
				},
			}

			Convey("when GenerateCustomTitle is called", func() {
				title := h.GenerateCustomTitle(dims)
				expected := "Label 1"
				So(title, ShouldResemble, expected)
			})
		})
	})
}
