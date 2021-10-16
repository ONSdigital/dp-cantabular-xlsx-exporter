package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	serviceMock "github.com/ONSdigital/dp-cantabular-xlsx-exporter/service/mock"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	//	dpkafka "github.com/ONSdigital/dp-kafka/v2"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/dp-kafka/v2/kafkatest"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	ctx           = context.Background()
	testBuildTime = "BuildTime"
	testGitCommit = "GitCommit"
	testVersion   = "Version"
)

var (
	errKafkaConsumer = fmt.Errorf("kafka consumer error")
	errHealthcheck   = fmt.Errorf("healthCheck error")
	errServer        = fmt.Errorf("HTTP Server error")
	errAddCheck      = fmt.Errorf("healthcheck add check error")
)

func TestInit(t *testing.T) {

	Convey("Having a set of mocked dependencies", t, func() {

		cfg, err := config.Get()
		So(err, ShouldBeNil)

		consumerMock := &kafkatest.IConsumerGroupMock{}
		GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
			return consumerMock, nil
		}

		producerMock := &kafkatest.IProducerMock{}
		GetKafkaProducer = func(ctx context.Context, cfg *config.Config) (kafka.IProducer, error) {
			return producerMock, nil
		}

		hcMock := &serviceMock.HealthCheckerMock{
			AddCheckFunc: func(name string, checker healthcheck.Checker) error { return nil },
		}
		GetHealthCheck = func(cfg *config.Config, buildTime, gitCommit, version string) (HealthChecker, error) {
			return hcMock, nil
		}

		serverMock := &serviceMock.HTTPServerMock{}
		GetHTTPServer = func(bindAddr string, router http.Handler) HTTPServer {
			return serverMock
		}

		s3UploaderMock := &serviceMock.S3UploaderMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}
		GetS3Uploader = func(cfg *config.Config) (S3Uploader, error) {
			return s3UploaderMock, nil
		}

		vaultMock := &serviceMock.VaultClientMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}
		GetVault = func(cfg *config.Config) (VaultClient, error) {
			return vaultMock, nil
		}

		GetProcessor = func(cfg *config.Config) Processor {
			return &serviceMock.ProcessorMock{
				ConsumeFunc: func(context.Context, kafka.IConsumerGroup, event.Handler) {},
			}
		}

		svc := &Service{}

		Convey("Given that initialising Kafka consumer returns an error", func() {
			GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
				return nil, errKafkaConsumer
			}

			Convey("Then service Init fails with the same error and no further initialisations are attempted", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(errors.Unwrap(err), ShouldResemble, errKafkaConsumer)
				So(svc.cfg, ShouldResemble, cfg)

				Convey("And no checkers are registered ", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 0)
				})
			})
		})

		Convey("Given that initialising healthcheck returns an error", func() {
			GetHealthCheck = func(cfg *config.Config, buildTime, gitCommit, version string) (HealthChecker, error) {
				return nil, errHealthcheck
			}

			Convey("Then service Init fails with the same error and no further initialisations are attempted", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(errors.Unwrap(err), ShouldResemble, errHealthcheck)
				So(svc.cfg, ShouldResemble, cfg)
				So(svc.consumer, ShouldResemble, consumerMock)

				Convey("And no checkers are registered ", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 0)
				})
			})
		})

		Convey("Given that Checkers cannot be registered", func() {
			hcMock.AddCheckFunc = func(name string, checker healthcheck.Checker) error { return errAddCheck }

			Convey("Then service Init fails with the expected error", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(err, ShouldNotBeNil)
				So(errors.Is(err, errAddCheck), ShouldBeTrue)
				So(svc.cfg, ShouldResemble, cfg)
				So(svc.consumer, ShouldResemble, consumerMock)

				Convey("And other checkers don't try to register", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 1)
				})
			})
		})

		Convey("Given that all dependencies are successfully initialised with encryption enabled", func() {
			cfg.EncryptionDisabled = false

			Convey("Then service Init succeeds, all dependencies are initialised", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(err, ShouldBeNil)
				So(svc.cfg, ShouldResemble, cfg)
				So(svc.server, ShouldEqual, serverMock)
				So(svc.healthCheck, ShouldResemble, hcMock)
				So(svc.consumer, ShouldResemble, consumerMock)
				//!!!				So(svc.cantabularClient, ShouldResemble, cantabularMock)
				So(svc.producer, ShouldResemble, producerMock)
				//!!!				So(svc.datasetAPIClient, ShouldResemble, datasetApiMock)
				//!!!				So(svc.cantabularClient, ShouldResemble, cantabularMock)
				So(svc.s3Uploader, ShouldResemble, s3UploaderMock)
				So(svc.vaultClient, ShouldResemble, vaultMock)

				Convey("And all checks are registered", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 6)
					So(hcMock.AddCheckCalls()[0].Name, ShouldResemble, "Kafka consumer")
					So(hcMock.AddCheckCalls()[1].Name, ShouldResemble, "Kafka producer")
					//					So(hcMock.AddCheckCalls()[2].Name, ShouldResemble, "Cantabular client")
					//					So(hcMock.AddCheckCalls()[3].Name, ShouldResemble, "Dataset API client")
					So(hcMock.AddCheckCalls()[4].Name, ShouldResemble, "S3 uploader")
					So(hcMock.AddCheckCalls()[5].Name, ShouldResemble, "Vault")
				})
			})
		})

		Convey("Given that all dependencies are successfully initialised with encryption disabled", func() {
			cfg.EncryptionDisabled = true

			Convey("Then service Init succeeds, all dependencies are initialised, except Vault", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(err, ShouldBeNil)
				So(svc.cfg, ShouldResemble, cfg)
				So(svc.server, ShouldEqual, serverMock)
				So(svc.healthCheck, ShouldResemble, hcMock)
				So(svc.consumer, ShouldResemble, consumerMock)
				//				So(svc.cantabularClient, ShouldResemble, cantabularMock)
				So(svc.producer, ShouldResemble, producerMock)
				//				So(svc.datasetAPIClient, ShouldResemble, datasetApiMock)
				//				So(svc.cantabularClient, ShouldResemble, cantabularMock)
				So(svc.s3Uploader, ShouldResemble, s3UploaderMock)

				Convey("And all checks are registered, except Vault", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 5)
					So(hcMock.AddCheckCalls()[0].Name, ShouldResemble, "Kafka consumer")
					So(hcMock.AddCheckCalls()[1].Name, ShouldResemble, "Kafka producer")
					//					So(hcMock.AddCheckCalls()[2].Name, ShouldResemble, "Cantabular client")
					//					So(hcMock.AddCheckCalls()[3].Name, ShouldResemble, "Dataset API client")
					So(hcMock.AddCheckCalls()[4].Name, ShouldResemble, "S3 uploader")
				})
			})
		})
	})
}

func TestStart(t *testing.T) {

	Convey("Having a correctly initialised Service with mocked dependencies", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)

		consumerMock := &kafkatest.IConsumerGroupMock{
			ChannelsFunc: func() *kafka.ConsumerGroupChannels { return &kafka.ConsumerGroupChannels{} },
		}

		processorMock := &serviceMock.ProcessorMock{
			ConsumeFunc: func(context.Context, kafka.IConsumerGroup, event.Handler) {},
		}

		hcMock := &serviceMock.HealthCheckerMock{
			StartFunc: func(ctx context.Context) {},
		}

		serverWg := &sync.WaitGroup{}
		serverMock := &serviceMock.HTTPServerMock{}

		svc := &Service{
			cfg:         cfg,
			server:      serverMock,
			healthCheck: hcMock,
			consumer:    consumerMock,
			processor:   processorMock,
		}

		Convey("When a service with a successful HTTP server is started", func() {
			serverMock.ListenAndServeFunc = func() error {
				serverWg.Done()
				return nil
			}
			serverWg.Add(1)
			svc.Start(ctx, make(chan error, 1))

			Convey("Then healthcheck is started and HTTP server starts listening", func() {
				So(len(hcMock.StartCalls()), ShouldEqual, 1)
				serverWg.Wait() // Wait for HTTP server go-routine to finish
				So(len(serverMock.ListenAndServeCalls()), ShouldEqual, 1)
			})
		})

		Convey("When a service with a failing HTTP server is started", func() {
			serverMock.ListenAndServeFunc = func() error {
				serverWg.Done()
				return errServer
			}
			errChan := make(chan error, 1)
			serverWg.Add(1)
			svc.Start(ctx, errChan)

			Convey("Then HTTP server errors are reported to the provided errors channel", func() {
				rxErr := <-errChan
				So(rxErr.Error(), ShouldResemble, fmt.Sprintf("failure in http listen and serve: %s", errServer.Error()))
			})
		})
	})
}

func TestClose(t *testing.T) {

	Convey("Having a correctly initialised service", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)

		hcStopped := false

		// kafka consumer mock
		consumerMock := &kafkatest.IConsumerGroupMock{
			StopListeningToConsumerFunc: func(ctx context.Context) error {
				return nil
			},
			CloseFunc: func(ctx context.Context) error { return nil },
		}

		// healthcheck Stop does not depend on any other service being closed/stopped
		hcMock := &serviceMock.HealthCheckerMock{
			StopFunc: func() { hcStopped = true },
		}

		// server Shutdown will fail if healthcheck is not stopped
		serverMock := &serviceMock.HTTPServerMock{
			ShutdownFunc: func(ctx context.Context) error {
				if !hcStopped {
					return fmt.Errorf("server stopped before healthcheck")
				}
				return nil
			},
		}

		svc := &Service{
			cfg:         cfg,
			server:      serverMock,
			healthCheck: hcMock,
			consumer:    consumerMock,
		}

		Convey("Closing the service results in all the dependencies being closed in the expected order", func() {
			err := svc.Close(context.Background())
			So(err, ShouldBeNil)
			So(consumerMock.StopListeningToConsumerCalls(), ShouldHaveLength, 1)
			So(hcMock.StopCalls(), ShouldHaveLength, 1)
			So(consumerMock.CloseCalls(), ShouldHaveLength, 1)
			So(serverMock.ShutdownCalls(), ShouldHaveLength, 1)
		})

		Convey("If services fail to stop, the Close operation tries to close all dependencies and returns an error", func() {
			consumerMock.StopListeningToConsumerFunc = func(ctx context.Context) error {
				return errKafkaConsumer
			}
			consumerMock.CloseFunc = func(ctx context.Context) error {
				return errKafkaConsumer
			}
			serverMock.ShutdownFunc = func(ctx context.Context) error {
				return errServer
			}

			err = svc.Close(context.Background())
			So(err, ShouldNotBeNil)
			So(consumerMock.StopListeningToConsumerCalls(), ShouldHaveLength, 1)
			So(consumerMock.CloseCalls(), ShouldHaveLength, 1)
			So(hcMock.StopCalls(), ShouldHaveLength, 1)
			So(serverMock.ShutdownCalls(), ShouldHaveLength, 1)
		})
	})
}
