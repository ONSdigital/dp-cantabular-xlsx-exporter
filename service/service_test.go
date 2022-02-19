package service_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/service"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	serviceMock "github.com/ONSdigital/dp-cantabular-xlsx-exporter/service/mock"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/dp-kafka/v3/kafkatest"

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

		consumerMock := &kafkatest.IConsumerGroupMock{
			RegisterHandlerFunc: func(ctx context.Context, h kafka.Handler) error {
				return nil
			},
		}
		service.GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
			return consumerMock, nil
		}

		hcMock := &serviceMock.HealthCheckerMock{
			AddCheckFunc:     func(name string, checker healthcheck.Checker) error { return nil },
			SubscribeAllFunc: func(s healthcheck.Subscriber) {},
		}
		service.GetHealthCheck = func(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
			return hcMock, nil
		}

		serverMock := &serviceMock.HTTPServerMock{}
		service.GetHTTPServer = func(bindAddr string, router http.Handler) service.HTTPServer {
			return serverMock
		}

		datasetApiMock := &serviceMock.DatasetAPIClientMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}

		// replace global function
		service.GetDatasetAPIClient = func(cfg *config.Config) service.DatasetAPIClient {
			return datasetApiMock
		}

		s3PrivateClientMock := &serviceMock.S3ClientMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}
		s3PublicClientMock := &serviceMock.S3ClientMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}
		service.GetS3Clients = func(cfg *config.Config) (service.S3Client, service.S3Client, error) {
			return s3PrivateClientMock, s3PublicClientMock, nil
		}

		vaultMock := &serviceMock.VaultClientMock{
			CheckerFunc: func(context.Context, *healthcheck.CheckState) error {
				return nil
			},
		}
		service.GetVault = func(cfg *config.Config) (service.VaultClient, error) {
			return vaultMock, nil
		}

		svc := &service.Service{}

		Convey("Given that initialising Kafka consumer returns an error", func() {
			service.GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
				return nil, errKafkaConsumer
			}

			Convey("Then service Init fails with the same error and no further initialisations are attempted", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(errors.Unwrap(err), ShouldResemble, errKafkaConsumer)
				So(svc.Cfg, ShouldResemble, cfg)

				Convey("And no checkers are registered ", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 0)
				})
			})
		})

		Convey("Given that initialising healthcheck returns an error", func() {
			service.GetHealthCheck = func(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
				return nil, errHealthcheck
			}

			Convey("Then service Init fails with the same error and no further initialisations are attempted", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(errors.Unwrap(err), ShouldResemble, errHealthcheck)
				So(svc.Cfg, ShouldResemble, cfg)
				So(svc.Consumer, ShouldResemble, consumerMock)

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
				So(svc.Cfg, ShouldResemble, cfg)
				So(svc.Consumer, ShouldResemble, consumerMock)

				Convey("And other checkers don't try to register", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 1)
				})
			})
		})

		Convey("Given that all dependencies are successfully initialised with encryption enabled", func() {

			Convey("Then service Init succeeds, all dependencies are initialised", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(err, ShouldBeNil)
				So(svc.Cfg, ShouldResemble, cfg)
				So(svc.Server, ShouldEqual, serverMock)
				So(svc.HealthCheck, ShouldResemble, hcMock)
				So(svc.Consumer, ShouldResemble, consumerMock)
				So(svc.DatasetAPIClient, ShouldResemble, datasetApiMock)
				So(svc.S3Private, ShouldResemble, s3PrivateClientMock)
				So(svc.S3Public, ShouldResemble, s3PublicClientMock)
				So(svc.VaultClient, ShouldResemble, vaultMock)

				Convey("And all checks are registered", func() {
					So(hcMock.AddCheckCalls(), ShouldHaveLength, 5)
					So(hcMock.AddCheckCalls()[0].Name, ShouldResemble, "Kafka consumer")
					So(hcMock.AddCheckCalls()[1].Name, ShouldResemble, "Dataset API client")
					So(hcMock.AddCheckCalls()[2].Name, ShouldResemble, "S3 private client")
					So(hcMock.AddCheckCalls()[3].Name, ShouldResemble, "S3 public client")
					So(hcMock.AddCheckCalls()[4].Name, ShouldResemble, "Vault")
				})

				Convey("And kafka consumer subscribes to all the healthcheck checkers", func() {
					So(hcMock.SubscribeAllCalls(), ShouldHaveLength, 1)
					So(hcMock.SubscribeAllCalls()[0].S, ShouldEqual, svc.Consumer)
				})

				Convey("And the kafka handler handler is registered to the consumer", func() {
					So(consumerMock.RegisterHandlerCalls(), ShouldHaveLength, 1)
				})
			})
		})

		Convey("Given that all dependencies are successfully initialised and StopConsumingOnUnhealthy is disabled", func() {
			cfg.StopConsumingOnUnhealthy = false
			defer func() {
				cfg.StopConsumingOnUnhealthy = true
			}()

			Convey("Then service Init succeeds, and the kafka consumer does not subscribe to the healthcheck library", func() {
				err := svc.Init(ctx, cfg, testBuildTime, testGitCommit, testVersion)
				So(err, ShouldBeNil)
				So(hcMock.SubscribeAllCalls(), ShouldHaveLength, 0)
			})
		})
	})
}

func TestStart(t *testing.T) {

	Convey("Having a correctly initialised Service with mocked dependencies", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)

		consumerMock := &kafkatest.IConsumerGroupMock{
			LogErrorsFunc: func(ctx context.Context) {},
			StartFunc:     func() error { return nil },
		}

		hcMock := &serviceMock.HealthCheckerMock{
			StartFunc: func(ctx context.Context) {},
		}

		serverWg := &sync.WaitGroup{}
		serverMock := &serviceMock.HTTPServerMock{}

		svc := &service.Service{
			Cfg:         cfg,
			Server:      serverMock,
			HealthCheck: hcMock,
			Consumer:    consumerMock,
		}

		Convey("When a service with a successful HTTP server is started", func() {
			serverMock.ListenAndServeFunc = func() error {
				serverWg.Done()
				return nil
			}
			serverWg.Add(1)
			err := svc.Start(ctx, make(chan error, 1))
			So(err, ShouldBeNil)

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
			err := svc.Start(ctx, errChan)
			So(err, ShouldBeNil)

			Convey("Then HTTP server errors are reported to the provided errors channel", func() {
				rxErr := <-errChan
				So(rxErr.Error(), ShouldResemble, fmt.Sprintf("failure in http listen and serve: %s", errServer.Error()))
			})
		})

		Convey("When a service with a successful HTTP server is started and StopConsumingOnUnhealthy is false", func() {
			cfg.StopConsumingOnUnhealthy = false
			defer func() {
				cfg.StopConsumingOnUnhealthy = true
			}()

			consumerMock.StartFunc = func() error { return nil }
			serverMock.ListenAndServeFunc = func() error {
				serverWg.Done()
				return nil
			}
			serverWg.Add(1)
			err := svc.Start(ctx, make(chan error, 1))
			So(err, ShouldBeNil)

			Convey("Then the kafka consumer is started", func() {
				So(consumerMock.StartCalls(), ShouldHaveLength, 1)
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
			StopFunc: func() error {
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

		svc := &service.Service{
			Cfg:         cfg,
			Server:      serverMock,
			HealthCheck: hcMock,
			Consumer:    consumerMock,
		}

		Convey("Closing the service results in all the dependencies being closed in the expected order", func() {
			err := svc.Close(ctx)
			So(err, ShouldBeNil)
			So(consumerMock.StopCalls(), ShouldHaveLength, 1)
			So(hcMock.StopCalls(), ShouldHaveLength, 1)
			So(consumerMock.CloseCalls(), ShouldHaveLength, 1)
			So(serverMock.ShutdownCalls(), ShouldHaveLength, 1)
		})

		Convey("If services fail to stop, the Close operation tries to close all dependencies and returns an error", func() {
			consumerMock.StopFunc = func() error {
				return nil
			}
			consumerMock.CloseFunc = func(ctx context.Context) error {
				return errKafkaConsumer
			}
			serverMock.ShutdownFunc = func(ctx context.Context) error {
				return errServer
			}

			err = svc.Close(ctx)
			So(err, ShouldNotBeNil)
			So(consumerMock.StopCalls(), ShouldHaveLength, 1)
			So(consumerMock.CloseCalls(), ShouldHaveLength, 1)
			So(hcMock.StopCalls(), ShouldHaveLength, 1)
			So(serverMock.ShutdownCalls(), ShouldHaveLength, 1)
		})
	})
}
