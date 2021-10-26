package handler_test

//!!! fix all of this code as the service implementation develops ...

/*	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"*/

//	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
//	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
//	"github.com/ONSdigital/dp-api-clients-go/v2/headers"
/*	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler/mock"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	"github.com/ONSdigital/dp-kafka/v2/kafkatest"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	. "github.com/smartystreets/goconvey/convey"*/

/*
const (
	testBucket             = "test-bucket"
	testVaultPath          = "vault-root"
	testInstanceID         = "test-instance-id"
	testS3Location         = "s3://myBucket/my-file.csv"
	testDownloadServiceURL = "http://test-download-service:8200"
	testETag               = "testETag"
)

var (
	testCsvBody = bufio.NewReader(bytes.NewReader([]byte("a,b,c,d,e,f,g,h,i,j,k,l")))
	testPsk     = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	errS3       = errors.New("test S3Upload error")
	errVault    = errors.New("test Vault error")
	errPsk      = errors.New("test PSK error")
	errDataset  = errors.New("test DatasetAPI error")
)

func testCfg() config.Config {
	return config.Config{
		UploadBucketName:   testBucket,
		VaultPath:          testVaultPath,
		EncryptionDisabled: true,
		DownloadServiceURL: testDownloadServiceURL,
	}
}

var ctx = context.Background()

// validateLines scans the provided reader, line by line, and compares with the corresponding line in the provided array.
// It also checks that all the expected lines are present in the reader.
func validateLines(reader *bufio.Reader, expectedLines []string) {
	i := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		So(scanner.Text(), ShouldEqual, expectedLines[i])
		i++
	}
	So(expectedLines, ShouldHaveLength, i) // Check that there aren't any more expected lines
	So(scanner.Err(), ShouldBeNil)
}

func TestUploadCSVFile(t *testing.T) {

	generator := &mock.GeneratorMock{
		NewPSKFunc: func() ([]byte, error) {
			return testPsk, nil
		},
	}

	isPublished := false

	expectedS3Key := fmt.Sprintf("instances/%s.csv", testInstanceID)
	expectedVaultPath := fmt.Sprintf("%s/%s.csv", testVaultPath, testInstanceID)

	Convey("Given an event handler with a successful S3Uploader", t, func() {
		s3Uploader := s3UploaderHappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, &s3Uploader, nil, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters and encryption disbled", func() {
			loc, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected location is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
			})

			Convey("Then the expected call Upload call is executed", func() {
				So(s3Uploader.UploadCalls(), ShouldHaveLength, 1)
				So(*s3Uploader.UploadCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*s3Uploader.UploadCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(s3Uploader.UploadCalls()[0].Input.Body, ShouldResemble, testCsvBody)
			})
		})
	})

	Convey("Given an event handler with a successful S3Uploader, Vault client and encryption enabled", t, func() {
		s3Uploader := s3UploaderHappy(true)
		vaultClient := vaultHappy()
		cfg := testCfg()
		cfg.EncryptionDisabled = false
		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &s3Uploader, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters", func() {
			loc, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected location is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
			})

			Convey("Then the expected key is stored in vault", func() {
				So(vaultClient.WriteKeyCalls(), ShouldHaveLength, 1)
				expectedPsk := hex.EncodeToString(testPsk)
				So(vaultClient.WriteKeyCalls()[0].Path, ShouldResemble, expectedVaultPath)
				So(vaultClient.WriteKeyCalls()[0].Key, ShouldResemble, "key")
				So(vaultClient.WriteKeyCalls()[0].Value, ShouldResemble, expectedPsk)
			})

			Convey("Then the expected call UploadWithPSK call is executed with the expected psk", func() {
				So(s3Uploader.UploadWithPSKCalls(), ShouldHaveLength, 1)
				expectedS3Key := fmt.Sprintf("instances/%s.csv", testInstanceID)
				So(*s3Uploader.UploadWithPSKCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*s3Uploader.UploadWithPSKCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(s3Uploader.UploadWithPSKCalls()[0].Input.Body, ShouldResemble, testCsvBody)
				So(s3Uploader.UploadWithPSKCalls()[0].Psk, ShouldResemble, testPsk)
			})
		})
	})

	Convey("Given an event handler with an unsuccessful S3Uploader", t, func() {
		s3Uploader := s3UploaderUnhappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, &s3Uploader, nil, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to upload unencrypted file to S3: %w", errS3),
					log.Data{
						"bucket":              testBucket,
						"filename":            fmt.Sprintf("instances/%s.csv", testInstanceID),
						"encryption_disabled": true,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler with an unsuccessful Vault client and encryption enabled", t, func() {
		cfg := testCfg()
		cfg.EncryptionDisabled = false
		s3Uploader := mock.S3UploaderMock{
			BucketNameFunc: func() string { return cfg.UploadBucketName },
		}
		vaultClient := vaultUnhappy()
		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &s3Uploader, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to write key to vault: %w", errVault),
					log.Data{
						"bucket":              testBucket,
						"filename":            fmt.Sprintf("instances/%s.csv", testInstanceID),
						"encryption_disabled": false,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an event handler with a successful Vault client, an unsuccessful S3 client and encryption enabled", t, func() {
		cfg := testCfg()
		cfg.EncryptionDisabled = false
		s3Uploader := s3UploaderUnhappy(true)
		vaultClient := vaultHappy()
		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &s3Uploader, &vaultClient, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to upload encrypted file to S3: %w", errS3),
					log.Data{
						"bucket":              testBucket,
						"filename":            fmt.Sprintf("instances/%s.csv", testInstanceID),
						"encryption_disabled": false,
						"is_published":        false,
					},
				))
			})
		})
	})

	Convey("Given an empty event handler", t, func() {
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, nil, nil, nil, nil)

		Convey("When UploadCSVFile is triggered with an empty instanceID", func() {
			_, err := eventHandler.UploadCSVFile(ctx, "", testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, errors.New("empty instance id not allowed"))
			})
		})

		Convey("When UploadCSVFile is triggered with a nil csv reader", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, nil, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, errors.New("no file content has been provided"))
			})
		})
	})

	Convey("Given an event handler, a failing createPSK function and encryption enabled", t, func() {
		cfg := testCfg()
		cfg.EncryptionDisabled = false
		s3Uploader := mock.S3UploaderMock{
			BucketNameFunc: func() string { return cfg.UploadBucketName },
		}

		generator.NewPSKFunc = func() ([]byte, error) {
			return nil, errPsk
		}

		eventHandler := handler.NewInstanceComplete(cfg, nil, nil, &s3Uploader, nil, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldResemble, fmt.Sprintf("failed to generate a PSK for encryption: %s", errPsk.Error()))
			})
		})
	})
}

func TestUploadPublishedCSVFile(t *testing.T) {

	generator := &mock.GeneratorMock{
		NewPSKFunc: func() ([]byte, error) {
			return testPsk, nil
		},
	}

	isPublished := true

	expectedS3Key := fmt.Sprintf("instances/%s.csv", testInstanceID)

	Convey("Given an event handler with a successful S3Uploader", t, func() {
		s3Uploader := s3UploaderHappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, &s3Uploader, nil, nil, generator)

		Convey("When UploadCSVFile is triggered with valid paramters and encryption disbled", func() {
			loc, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected location is returned with no error ", func() {
				So(err, ShouldBeNil)
				So(loc, ShouldEqual, testS3Location)
			})

			Convey("Then the expected call Upload call is executed", func() {
				So(s3Uploader.UploadCalls(), ShouldHaveLength, 1)
				So(*s3Uploader.UploadCalls()[0].Input.Key, ShouldResemble, expectedS3Key)
				So(*s3Uploader.UploadCalls()[0].Input.Bucket, ShouldResemble, testBucket)
				So(s3Uploader.UploadCalls()[0].Input.Body, ShouldResemble, testCsvBody)
			})
		})
	})

	Convey("Given an event handler with an unsuccessful S3Uploader", t, func() {
		s3Uploader := s3UploaderUnhappy(false)
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, &s3Uploader, nil, nil, generator)

		Convey("When UploadCSVFile is triggered", func() {
			_, err := eventHandler.UploadCSVFile(ctx, testInstanceID, testCsvBody, isPublished)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, handler.NewError(
					fmt.Errorf("failed to upload published file to S3: %w", errS3),
					log.Data{
						"bucket":       testBucket,
						"filename":     fmt.Sprintf("instances/%s.csv", testInstanceID),
						"is_published": true,
					},
				))
			})
		})
	})
}

func TestUpdateInstance(t *testing.T) {
	testSize := testCsvBody.Size()

	Convey("Given an event handler with a successful dataset API mock", t, func() {
		datasetAPIMock := datasetAPIClientHappy()
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, &datasetAPIMock, nil, nil, nil, nil)

		Convey("When UpdateInstance is called", func() {
			err := eventHandler.UpdateInstance(ctx, testInstanceID, testSize)

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the expected UpdateInstance call is executed with the expected paramters", func() {
				expectedURL := fmt.Sprintf("%s/downloads/instances/%s.csv", testDownloadServiceURL, testInstanceID)
				So(datasetAPIMock.GetInstanceCalls(), ShouldHaveLength, 0)
				So(datasetAPIMock.PutInstanceCalls(), ShouldHaveLength, 1)
				So(datasetAPIMock.PutInstanceCalls()[0].InstanceID, ShouldEqual, testInstanceID)
				So(datasetAPIMock.PutInstanceCalls()[0].InstanceUpdate, ShouldResemble, dataset.UpdateInstance{
					Downloads: dataset.DownloadList{
						CSV: &dataset.Download{
							URL:  expectedURL,
							Size: fmt.Sprintf("%d", testSize),
						},
					},
				})
				So(datasetAPIMock.PutInstanceCalls()[0].IfMatch, ShouldEqual, headers.IfMatchAnyETag)
			})
		})
	})

	Convey("Given an event handler with a failing dataset API mock", t, func() {
		datasetAPIMock := datasetAPIClientUnhappy()
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, &datasetAPIMock, nil, nil, nil, nil)

		Convey("When UpdateInstance is called", func() {
			err := eventHandler.UpdateInstance(ctx, testInstanceID, testSize)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, fmt.Errorf("error during put instance: %w", errDataset))
			})
		})
	})
}

func TestProduceExportCompleteEvent(t *testing.T) {
	expectedEvent := event.CantabularCsvCreated{ //!!! probably should be 'InstanceComplete'
		FileURL:    fmt.Sprintf("%s/downloads/instances/%s.csv", testDownloadServiceURL, testInstanceID),
		InstanceID: testInstanceID,
	}

	Convey("Given an event handler with a successful Kafka Producer", t, func(c C) {
		producer := kafkatest.NewMessageProducer(true)
		eventHandler := handler.NewInstanceComplete(testCfg(), nil, nil, nil, nil, producer, nil)

		Convey("When ProduceExportCompleteEvent is called", func(c C) {
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := eventHandler.ProduceExportCompleteEvent(testInstanceID)
				c.So(err, ShouldBeNil)
			}()

			Convey("Then the expected message is produced", func() {
				producedBytes := <-producer.Channels().Output
				producedMessage := event.CantabularCsvCreated{} //!!! probably should be 'InstanceComplete'
				err := schema.CantabularCsvCreated.Unmarshal(producedBytes, &producedMessage) //!!! probably should be 'InstanceComplete'
				So(err, ShouldBeNil)
				So(producedMessage, ShouldResemble, expectedEvent)
			})

			// make sure the go-routine finishes its execution
			wg.Wait()
		})
	})
}

func s3UploaderHappy(encryptionEnabled bool) mock.S3UploaderMock {
	if encryptionEnabled {
		return mock.S3UploaderMock{
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
	return mock.S3UploaderMock{
		UploadFunc: func(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
			return &s3manager.UploadOutput{
				Location: testS3Location,
			}, nil
		},
		BucketNameFunc: func() string {
			return testBucket
		},
	}
}

func vaultHappy() mock.VaultClientMock {
	return mock.VaultClientMock{
		WriteKeyFunc: func(path string, key string, value string) error {
			return nil
		},
	}
}

func vaultUnhappy() mock.VaultClientMock {
	return mock.VaultClientMock{
		WriteKeyFunc: func(path string, key string, value string) error {
			return errVault
		},
	}
}

func s3UploaderUnhappy(encryptionEnabled bool) mock.S3UploaderMock {
	if encryptionEnabled {
		return mock.S3UploaderMock{
			UploadWithPSKFunc: func(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error) {
				return nil, errS3
			},
			BucketNameFunc: func() string {
				return testBucket
			},
		}
	}
	return mock.S3UploaderMock{
		UploadFunc: func(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
			return nil, errS3
		},
		BucketNameFunc: func() string {
			return testBucket
		},
	}
}

func datasetAPIClientHappy() mock.DatasetAPIClientMock {
	return mock.DatasetAPIClientMock{
		GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
			return dataset.Instance{}, "", nil
		},
		PutInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, instanceUpdate dataset.UpdateInstance, ifMatch string) (string, error) {
			return testETag, nil
		},
	}
}

func datasetAPIClientUnhappy() mock.DatasetAPIClientMock {
	return mock.DatasetAPIClientMock{
		GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
			return dataset.Instance{}, "", errDataset
		},
		PutInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, instanceUpdate dataset.UpdateInstance, ifMatch string) (string, error) {
			return "", errDataset
		},
	}
}
*/
