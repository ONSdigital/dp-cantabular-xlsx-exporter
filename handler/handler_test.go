package handler_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
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
	testS3Location         = "s3://myBucket/my-file.csv"
	testDownloadServiceURL = "http://test-download-service:8200"
	testNumBytes           = 123
	testRowCount           = 18
)

var (
	testExportStartEvent = &event.CantabularCsvCreated{
		InstanceID: testInstanceID,
		DatasetID:  testDatasetID,
		Edition:    testEdition,
		Version:    testVersion,
	}

	testCsvBody = bufio.NewReader(bytes.NewReader([]byte("a,b,c,d,e,f,g,h,i,j,k,l")))
	testPsk     = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	testReq     = cantabular.StaticDatasetQueryRequest{
		Dataset:   "Example",
		Variables: []string{"city", "siblings"},
	}
	errCantabular = errors.New("test Cantabular error")
	errS3         = errors.New("test S3Upload error")
	errVault      = errors.New("test Vault error")
	errPsk        = errors.New("test PSK error")
	errDataset    = errors.New("test DatasetAPI error")
)

func testCfg() config.Config {
	return config.Config{
		PublicBucketName:   testBucket,
		VaultPath:          testVaultPath,
		DownloadServiceURL: testDownloadServiceURL,
	}
}

var ctx = context.Background()

/* !!! sort this out
func TestValidateInstance(t *testing.T) {
	h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, nil, nil, nil)

	Convey("Given an instance that is in 'published' state", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				ID:    "myID",
				State: dataset.StatePublished.String(),
			},
		}
		Convey("Then we see the instance is published", func() {
			isPublished, err := h.ValidateInstance(i)
			So(err, ShouldBeNil)
			So(isPublished, ShouldBeTrue)
		})
	})

	Convey("Given an instance with 2 CSV headers in 'associated' state", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				CSVHeader: []string{"1", "2"},
				IsBasedOn: &dataset.IsBasedOn{ID: "myID"},
				State:     dataset.StateAssociated.String(),
			},
		}
		Convey("Then ValidateInstance determines that instance is not published, without error", func() {
			isPublished, err := h.ValidateInstance(i)
			So(err, ShouldBeNil)
			So(isPublished, ShouldBeFalse)
		})
	})

	Convey("Given an instance with 2 CSV headers and no state", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				CSVHeader: []string{"1", "2"},
				IsBasedOn: &dataset.IsBasedOn{ID: "myID"},
			},
		}
		Convey("Then ValidateInstance determines that instance is not published, without error", func() {
			isPublished, err := h.ValidateInstance(i)
			So(err, ShouldBeNil)
			So(isPublished, ShouldBeFalse)
		})
	})

	Convey("Given an instance wit only 1 CSV header", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				CSVHeader: []string{"1"},
				IsBasedOn: &dataset.IsBasedOn{ID: "myID"},
			},
		}
		Convey("Then ValidateInstance returns the expected error", func() {
			_, err := h.ValidateInstance(i)
			So(err, ShouldResemble, handler.NewError(
				errors.New("no dimensions in headers"),
				log.Data{"headers": []string{"1"}},
			))
		})
	})

	Convey("Given an instance without isBasedOn field", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				CSVHeader: []string{"1", "2"},
				State:     dataset.StatePublished.String(),
			},
		}
		Convey("Then ValidateInstance returns the expected error", func() {
			_, err := h.ValidateInstance(i)
			So(err, ShouldResemble, handler.NewError(
				errors.New("missing instance isBasedOn.ID"),
				log.Data{"is_based_on": (*dataset.IsBasedOn)(nil)},
			))
		})
	})

	Convey("Given an instance with an empty isBasedOn field", t, func() {
		i := dataset.Instance{
			Version: dataset.Version{
				CSVHeader: []string{"1", "2"},
				State:     dataset.StatePublished.String(),
				IsBasedOn: &dataset.IsBasedOn{},
			},
		}
		Convey("Then ValidateInstance returns the expected error", func() {
			_, err := h.ValidateInstance(i)
			So(err, ShouldResemble, handler.NewError(
				errors.New("missing instance isBasedOn.ID"),
				log.Data{"is_based_on": &dataset.IsBasedOn{}},
			))
		})
	})
}
*/

/*
func TestUploadPrivateUnEncryptedCSVFile(t *testing.T) {
	isPublished := false
	expectedS3Key := fmt.Sprintf("datasets/%s-%s-%s.csv", testDatasetID, testEdition, testVersion)

	Convey("Given an event handler with a successful cantabular client and private S3Client", t, func() {
		c := cantabularMock(testCsvBody)
		sPrivate := s3ClientHappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), &c, nil, &sPrivate, nil, nil, nil, nil)

		Convey("When UploadCSVFile is triggered with valid paramters and encryption disbled", func() {
			loc, rowCount, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected location and rowCount is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
				So(rowCount, ShouldEqual, testRowCount)
			})

			Convey("Then the expected UploadWithContext call is executed", func() {
				So(sPrivate.UploadWithContextCalls(), ShouldHaveLength, 1)
				So(sPrivate.UploadWithContextCalls()[0].Ctx, ShouldResemble, ctx)
				So(*sPrivate.UploadWithContextCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*sPrivate.UploadWithContextCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(sPrivate.UploadWithContextCalls()[0].Input.Body, ShouldResemble, testCsvBody)
			})
		})
	})

	Convey("Given an event handler with a successful cantabular client and an unsuccessful private S3Client", t, func() {
		c := cantabularMock(testCsvBody)
		sPrivate := s3ClientUnhappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), &c, nil, &sPrivate, nil, nil, nil, nil)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w",
						fmt.Errorf("failed to upload un-encrypted private file to S3: %w", errS3),
					),
					log.Data{
						"bucket":              testBucket,
						"filename":            expectedS3Key,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler with an unsuccessful cantabular client", t, func() {
		c := cantabularUnhappy()
		sPrivate := mock.S3ClientMock{
			BucketNameFunc: func() string { return testBucket },
		}
		eventHandler := handler.NewInstanceComplete(testCfg(), &c, nil, &sPrivate, nil, nil, nil, nil)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w", errCantabular),
					log.Data{
						"bucket":              testBucket,
						"filename":            expectedS3Key,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an empty event handler", t, func() {
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, nil, nil, nil, nil, nil)

		Convey("When UploadCSVFile is triggered with an empty export-start event", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, &event.ExportStart{}, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, errors.New("empty instance id not allowed"))
			})
		})
	})
}
*/

/*
func TestUploadPrivateEncryptedCSVFile(t *testing.T) {
	generator := &mock.GeneratorMock{
		NewPSKFunc: func() ([]byte, error) {
			return testPsk, nil
		},
	}
	isPublished := false
	expectedS3Key := fmt.Sprintf("datasets/%s-%s-%s.csv", testDatasetID, testEdition, testVersion)
	expectedVaultPath := fmt.Sprintf("%s/%s-%s-%s.csv", testVaultPath, testDatasetID, testEdition, testVersion)
	cfg := testCfg()

	Convey("Given an event handler with a successful cantabular client, private S3Client, Vault client and encryption enabled", t, func() {
		c := cantabularMock(testCsvBody)
		sPrivate := s3ClientHappy(true)
		v := vaultHappy()
		eventHandler := handler.NewInstanceComplete(cfg, &c, nil, &sPrivate, nil, &v, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters", func() {
			loc, rowCount, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected location and rowCount is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
				So(rowCount, ShouldEqual, testRowCount)
			})

			Convey("Then the expected key is stored in vault", func() {
				So(v.WriteKeyCalls(), ShouldHaveLength, 1)
				expectedPsk := hex.EncodeToString(testPsk)
				So(v.WriteKeyCalls()[0].Path, ShouldResemble, expectedVaultPath)
				So(v.WriteKeyCalls()[0].Key, ShouldResemble, "key")
				So(v.WriteKeyCalls()[0].Value, ShouldResemble, expectedPsk)
			})

			Convey("Then the expected call UploadWithPSK call is executed with the expected psk", func() {
				So(sPrivate.UploadWithPSKCalls(), ShouldHaveLength, 1)
				So(*sPrivate.UploadWithPSKCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*sPrivate.UploadWithPSKCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(sPrivate.UploadWithPSKCalls()[0].Input.Body, ShouldResemble, testCsvBody)
				So(sPrivate.UploadWithPSKCalls()[0].Psk, ShouldResemble, testPsk)
			})
		})
	})

	Convey("Given an event handler with an unsuccessful Vault client and encryption enabled", t, func() {
		sPrivate := mock.S3ClientMock{
			BucketNameFunc: func() string { return testBucket },
		}
		vaultClient := vaultUnhappy()
		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &sPrivate, nil, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to write key to vault: %w", errVault),
					log.Data{
						"bucket":              testBucket,
						"filename":            expectedS3Key,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler with a successful Vault client, a successful cantabular client, an unsuccessful private S3 client and encryption enabled", t, func() {
		c := cantabularMock(testCsvBody)
		sPrivate := s3ClientUnhappy(true)
		vaultClient := vaultHappy()
		eventHandler := handler.NewInstanceComplete(cfg, &c, nil, &sPrivate, nil, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w",
						fmt.Errorf("failed to upload encrypted private file to S3: %w", errS3),
					),
					log.Data{
						"bucket":              testBucket,
						"filename":            expectedS3Key,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler with a successful Vault client, an unsuccessful cantabular client and encryption enabled", t, func() {
		c := cantabularUnhappy()
		sPrivate := mock.S3ClientMock{
			BucketNameFunc: func() string { return testBucket },
		}
		vaultClient := vaultHappy()
		eventHandler := handler.NewInstanceComplete(cfg, &c, nil, &sPrivate, nil, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w", errCantabular),
					log.Data{
						"bucket":              testBucket,
						"filename":            expectedS3Key,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler, a failing createPSK function and encryption enabled", t, func() {
		sPrivate := mock.S3ClientMock{
			BucketNameFunc: func() string { return testBucket },
		}

		generator.NewPSKFunc = func() ([]byte, error) {
			return nil, errPsk
		}

		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &sPrivate, nil, nil, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldResemble, fmt.Sprintf("failed to generate a PSK for encryption: %s", errPsk.Error()))
			})
		})
	})
}
*/

/*
func TestUploadPublishedCSVFile(t *testing.T) {
	isPublished := true
	expectedS3Key := fmt.Sprintf("datasets/%s-%s-%s.csv", testDatasetID, testEdition, testVersion)

	Convey("Given an event handler with a successful cantabular client and public S3Client", t, func() {
		c := cantabularMock(testCsvBody)
		sPublic := s3ClientHappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), &c, nil, nil, &sPublic, nil, nil, nil)

		Convey("When UploadCSVFile is triggered with valid paramters", func() {
			loc, rowCount, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected location and rowCount is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
				So(rowCount, ShouldEqual, testRowCount)
			})

			Convey("Then the expected UploadWithContext call is executed", func() {
				So(sPublic.UploadWithContextCalls(), ShouldHaveLength, 1)
				So(sPublic.UploadWithContextCalls()[0].Ctx, ShouldResemble, ctx)
				So(*sPublic.UploadWithContextCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*sPublic.UploadWithContextCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(sPublic.UploadWithContextCalls()[0].Input.Body, ShouldResemble, testCsvBody)
			})
		})
	})

	Convey("Given an event handler with a successful cantabular client and an unsuccessful public S3Client", t, func() {
		c := cantabularMock(testCsvBody)
		publicS3Client := s3ClientUnhappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), &c, nil, nil, &publicS3Client, nil, nil, nil)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w",
						fmt.Errorf("failed to upload published file to S3: %w", errS3),
					),
					log.Data{
						"bucket":       testBucket,
						"filename":     expectedS3Key,
						"is_published": true,
					}),
				)
			})
		})
	})

	Convey("Given an event handler with an unsuccessful cantabular client", t, func() {
		cfg := testCfg()
		c := cantabularUnhappy()
		publicS3Client := mock.S3ClientMock{
			BucketNameFunc: func() string { return testBucket },
		}
		eventHandler := handler.NewInstanceComplete(cfg, &c, nil, nil, &publicS3Client, nil, nil, nil)

		Convey("When UploadCSVFile is triggered", func() {
			_, _, err := eventHandler.UploadCSVFile(ctx, testExportStartEvent, isPublished, testReq)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to stream csv data: %w", errCantabular),
					log.Data{
						"bucket":       testBucket,
						"filename":     expectedS3Key,
						"is_published": true,
					}),
				)
			})
		})
	})
}
*/

//!!! sort this
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
			So(err, ShouldResemble, fmt.Errorf("private s3 head object error: %w", errS3))
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
			So(err, ShouldResemble, fmt.Errorf("public s3 head object error: %w", errS3))
		})
	})
}

/*!!! sort this
func TestUpdateInstance(t *testing.T) {
	testSize := testCsvBody.Size()

	Convey("Given an event handler with a successful dataset API mock", t, func() {
		datasetAPIMock := datasetAPIClientHappy()
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("When UpdateInstance is called for a private csv file", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, false, "")

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the expected UpdateInstance call is executed with the expected paramters", func() {
				expectedURL := fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.csv", testDownloadServiceURL, testDatasetID, testEdition, testVersion)
				So(datasetAPIMock.GetInstanceCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.PutVersionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.PutVersionCalls()[0].DatasetID, ShouldEqual, testDatasetID)
				So(datasetAPIMock.PutVersionCalls()[0].Edition, ShouldEqual, testEdition)
				So(datasetAPIMock.PutVersionCalls()[0].Version, ShouldEqual, testVersion)
				So(datasetAPIMock.PutVersionCalls()[0].V, ShouldResemble, dataset.Version{
					Downloads: map[string]dataset.Download{
						"CSV": {
							URL:  expectedURL,
							Size: fmt.Sprintf("%d", testSize),
						},
					},
				})
			})
		})

		Convey("When UpdateInstance is called for a public csv file", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, true, "publicURL")

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the expected UpdateInstance call is executed once to update the public download link and associate it, and once more to publish it", func() {
				expectedURL := fmt.Sprintf("%s/downloads/datasets/%s/editions/%s/versions/%s.csv", testDownloadServiceURL, testDatasetID, testEdition, testVersion)
				So(datasetAPIMock.GetInstanceCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.PutVersionCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.PutVersionCalls()[0].DatasetID, ShouldEqual, testDatasetID)
				So(datasetAPIMock.PutVersionCalls()[0].Edition, ShouldEqual, testEdition)
				So(datasetAPIMock.PutVersionCalls()[0].Version, ShouldEqual, testVersion)
				So(datasetAPIMock.PutVersionCalls()[0].V, ShouldResemble, dataset.Version{
					Downloads: map[string]dataset.Download{
						"CSV": {
							Public: "publicURL",
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
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, &datasetAPIMock, nil, nil, nil, nil, nil)

		Convey("When UpdateInstance is called", func() {
			err := eventHandler.UpdateInstance(ctx, testExportStartEvent, testSize, false, "")

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, fmt.Errorf("error while attempting update version downloads: %w", errDataset))
			})
		})
	})
}
*/

/*!!! sort this
func s3ClientHappy(encryptionEnabled bool) mock.S3ClientMock {
	if encryptionEnabled {
		return mock.S3ClientMock{
			UploadWithPSKFunc: func(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error) {
				return &s3manager.UploadOutput{
					Location: testS3Location,
				}, nil
			},
			BucketNameFunc: func() string {
				return testBucket
			},
		}
	}
	return mock.S3ClientMock{
		UploadWithContextFunc: func(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
			return &s3manager.UploadOutput{
				Location: testS3Location,
			}, nil
		},
		BucketNameFunc: func() string {
			return testBucket
		},
	}
}
*/

/*!!! sort this
func vaultHappy() mock.VaultClientMock {
	return mock.VaultClientMock{
		WriteKeyFunc: func(path string, key string, value string) error {
			return nil
		},
	}
}
*/

/*!!! sort this
func vaultUnhappy() mock.VaultClientMock {
	return mock.VaultClientMock{
		WriteKeyFunc: func(path string, key string, value string) error {
			return errVault
		},
	}
}
*/

/*!!! sort this
func s3ClientUnhappy(encryptionEnabled bool) mock.S3ClientMock {
	if encryptionEnabled {
		return mock.S3ClientMock{
			UploadWithPSKFunc: func(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error) {
				return nil, errS3
			},
			BucketNameFunc: func() string {
				return testBucket
			},
		}
	}
	return mock.S3ClientMock{
		UploadWithContextFunc: func(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
			return nil, errS3
		},
		BucketNameFunc: func() string {
			return testBucket
		},
	}
}
*/

/*!!! sort this
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
*/

/*!!! sort this
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
*/
