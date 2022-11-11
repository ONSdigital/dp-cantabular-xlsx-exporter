package handler_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler/mock"
	"github.com/aws/aws-sdk-go/service/s3"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testBucket             = "test-bucket"
	testVaultPath          = "vault-root"
	testInstanceID         = "test-instance-id"
	testDatasetID          = "test-dataset-id"
	testEdition            = "test-edition"
	testVersion            = "test-version"
	testDownloadServiceURL = "http://test-download-service:8200"
	testNumBytes           = 123
	testS3PublicURL        = "test-bucket"
	testFileName           = "datasets/test-version.xlsx"
)

var (
	testExportStartEvent = &event.CantabularCsvCreated{
		InstanceID: testInstanceID,
		DatasetID:  testDatasetID,
		Edition:    testEdition,
		Version:    testVersion,
	}

	testCsvBody = bufio.NewReader(bytes.NewReader([]byte("a,b,c,d,e,f,g,h,i,j,k,l")))
	errS3       = errors.New("test S3Upload error")
	errDataset  = errors.New("test DatasetAPI error")
)

func testCfg() config.Config {
	return config.Config{
		PublicBucketName:   testBucket,
		VaultPath:          testVaultPath,
		DownloadServiceURL: testDownloadServiceURL,
	}
}

var ctx = context.Background()

func TestIsInstancePublished(t *testing.T) {

	Convey("Given a dataset api that fails to get the instance", t, func() {
		datasetAPIMock := mock.DatasetAPIClientMock{
			GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
				return dataset.Instance{}, "", errors.New("dataset api error")
			},
		}
		h := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("Then we see the expected error", func() {
			isPublished, err := h.IsInstancePublished(ctx, "testID")
			So(err.Error(), ShouldContainSubstring, "dataset api error")
			So(isPublished, ShouldBeTrue)
		})
	})

	Convey("Given a dataset api that gets an instance for published instance", t, func() {
		datasetAPIMock := mock.DatasetAPIClientMock{
			GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
				return dataset.Instance{
					Version: dataset.Version{
						State: dataset.StatePublished.String(),
					},
				}, "", nil
			},
		}
		h := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("Then we see the expected value of true", func() {
			isPublished, err := h.IsInstancePublished(ctx, "testID")
			So(err, ShouldBeNil)
			So(isPublished, ShouldBeTrue)
		})
	})

	Convey("Given a dataset api that gets an instance for non published instance", t, func() {
		datasetAPIMock := mock.DatasetAPIClientMock{
			GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
				return dataset.Instance{
					Version: dataset.Version{
						State: dataset.StateAssociated.String(),
					},
				}, "", nil
			},
		}
		h := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("Then we see the expected value of false", func() {
			isPublished, err := h.IsInstancePublished(ctx, "testID")
			So(err, ShouldBeNil)
			So(isPublished, ShouldBeFalse)
		})
	})
}

func TestValidateEvent(t *testing.T) {
	Convey("Given an event that is good", t, func() {
		kafkaGoodEvent := &event.CantabularCsvCreated{
			InstanceID: "good",
			RowCount:   10,
		}

		err := handler.ValidateEvent(kafkaGoodEvent)
		Convey("Then ValidateEvent returns with no error", func() {
			So(err, ShouldBeNil)
		})
	})

	Convey("Given an event where RowCount is greater than 'MaxAllowedRowCount'", t, func() {
		kafkaBadMaxRowEvent := &event.CantabularCsvCreated{
			RowCount: handler.MaxAllowedRowCount + 10,
		}

		err := handler.ValidateEvent(kafkaBadMaxRowEvent)
		Convey("Then ValidateEvent returns error", func() {
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Given an event without an InstanceID", t, func() {
		kafkaBadInstanceIDEvent := &event.CantabularCsvCreated{
			RowCount: 10,
		}

		err := handler.ValidateEvent(kafkaBadInstanceIDEvent)
		Convey("Then ValidateEvent returns error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetS3ContentLength(t *testing.T) {
	var ContentLength int64 = testNumBytes
	headOk := func(key string) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{
			ContentLength: &ContentLength,
		}, nil
	}
	headErr := func(key string) (*s3.HeadObjectOutput, error) {
		return nil, errS3
	}

	Convey("Given an event handler with a successful s3 private client", t, func() {
		sPrivate := mock.S3ClientMock{HeadFunc: headOk}
		eventHandler := handler.NewXlsxCreate(testCfg(), nil, &sPrivate, nil, nil, nil, nil)
		Convey("Then GetS3ContentLength returns the expected size with no error", func() {
			numBytes, err := eventHandler.GetS3ContentLength(testExportStartEvent, false)
			So(err, ShouldBeNil)
			So(numBytes, ShouldEqual, testNumBytes)
		})
	})

	Convey("Given an event handler with a failing s3 private client", t, func() {
		sPrivate := mock.S3ClientMock{HeadFunc: headErr}
		eventHandler := handler.NewXlsxCreate(testCfg(), nil, &sPrivate, nil, nil, nil, nil)

		Convey("Then GetS3ContentLength returns the expected error", func() {
			_, err := eventHandler.GetS3ContentLength(testExportStartEvent, false)
			So(err.Error(), ShouldContainSubstring, "test S3Upload error")
		})
	})

	Convey("Given an event handler with a successful s3 public client", t, func() {
		sPublic := mock.S3ClientMock{HeadFunc: headOk}
		eventHandler := handler.NewXlsxCreate(testCfg(), nil, nil, &sPublic, nil, nil, nil)

		Convey("Then GetS3ContentLength returns the expected size with no error", func() {
			numBytes, err := eventHandler.GetS3ContentLength(testExportStartEvent, true)
			So(err, ShouldBeNil)
			So(numBytes, ShouldEqual, testNumBytes)
		})
	})

	Convey("Given an event handler with a failing s3 public client", t, func() {
		sPublic := mock.S3ClientMock{HeadFunc: headErr}
		eventHandler := handler.NewXlsxCreate(testCfg(), nil, nil, &sPublic, nil, nil, nil)

		Convey("Then GetS3ContentLength returns the expected error", func() {
			_, err := eventHandler.GetS3ContentLength(testExportStartEvent, true)
			So(err.Error(), ShouldContainSubstring, "test S3Upload error")
		})
	})
}

func TestUpdateInstance(t *testing.T) {
	testSize := testCsvBody.Size()

	Convey("Given an event handler with a successful dataset API mock", t, func() {
		datasetAPIMock := datasetAPIClientHappy()
		eventHandler := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("When UpdateInstance is called for a private csv file", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, false, "", testFileName)

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the expected UpdateInstance call is executed with the expected parameters", func() {
				expectedURL := fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.xlsx", testDownloadServiceURL, testDatasetID, testEdition, testVersion)
				So(datasetAPIMock.GetInstanceCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.PutVersionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.PutVersionCalls()[0].DatasetID, ShouldEqual, testDatasetID)
				So(datasetAPIMock.PutVersionCalls()[0].Edition, ShouldEqual, testEdition)
				So(datasetAPIMock.PutVersionCalls()[0].Version, ShouldEqual, testVersion)
				So(datasetAPIMock.PutVersionCalls()[0].M, ShouldResemble, dataset.Version{
					Downloads: map[string]dataset.Download{
						"XLS": {
							URL:  expectedURL,
							Size: fmt.Sprintf("%d", testSize),
						},
					},
				})
			})
		})

		Convey("When UpdateInstance is called for a public xlsx file", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, true, "publicURL", testFileName)

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the expected UpdateInstance call is executed once to update the public download link and associate it, and once more to publish it", func() {
				expectedURL := fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.xlsx", testDownloadServiceURL, testDatasetID, testEdition, testVersion)
				So(datasetAPIMock.GetInstanceCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.PutVersionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.PutVersionCalls()[0].DatasetID, ShouldEqual, testDatasetID)
				So(datasetAPIMock.PutVersionCalls()[0].Edition, ShouldEqual, testEdition)
				So(datasetAPIMock.PutVersionCalls()[0].Version, ShouldEqual, testVersion)
				So(datasetAPIMock.PutVersionCalls()[0].M, ShouldResemble, dataset.Version{
					Downloads: map[string]dataset.Download{
						"XLS": {
							Public: fmt.Sprintf("/datasets/%s.xlsx", testVersion),
							URL:    expectedURL,
							Size:   fmt.Sprintf("%d", testSize),
						},
					},
				})
			})
		})
	})

	Convey("Given an event handler with a failing dataset API mock", t, func() {
		datasetAPIMock := datasetAPIClientUnhappy()
		eventHandler := handler.NewXlsxCreate(testCfg(), &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("When UpdateInstance is called", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, false, "", testFileName)

			Convey("Then the expected error is returned", func() {
				So(err.Error(), ShouldContainSubstring, "error while attempting update version downloads")
			})
		})
	})
}

func datasetAPIClientHappy() mock.DatasetAPIClientMock {
	return mock.DatasetAPIClientMock{
		GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
			return dataset.Instance{}, "", nil
		},
		PutVersionFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, datasetID string, edition string, version string, v dataset.Version) error {
			return nil
		},
	}
}

func datasetAPIClientUnhappy() mock.DatasetAPIClientMock {
	return mock.DatasetAPIClientMock{
		GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
			return dataset.Instance{}, "", errDataset
		},
		PutVersionFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, datasetID string, edition string, version string, v dataset.Version) error {
			return errDataset
		},
	}
}
