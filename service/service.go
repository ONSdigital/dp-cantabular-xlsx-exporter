package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	dps3 "github.com/ONSdigital/dp-s3"
	"github.com/ONSdigital/log.go/v2/log"

	"github.com/gorilla/mux"
)

// Service contains all the configs, server and clients to run the event handler service
type Service struct {
	Cfg                 *config.Config
	Server              HTTPServer
	HealthCheck         HealthChecker
	Consumer            kafka.IConsumerGroup
	Producer            kafka.IProducer
	DatasetAPIClient    DatasetAPIClient
	Processor           Processor
	S3PrivateUploader   S3Client
	S3PublicUploader    S3Client
	S3PrivateDownloader *dps3.S3
	S3PublicDownloader  *dps3.S3
	VaultClient         VaultClient
	generator           Generator
}

func New() *Service {
	return &Service{}
}

// Init initialises the service and it's dependencies
func (svc *Service) Init(ctx context.Context, cfg *config.Config, buildTime, gitCommit, version string) error {
	var err error

	if cfg == nil {
		return errors.New("nil config passed to service init")
	}

	svc.Cfg = cfg

	if svc.Consumer, err = GetKafkaConsumer(ctx, cfg); err != nil {
		return fmt.Errorf("failed to create kafka consumer: %w", err)
	}
	if svc.Producer, err = GetKafkaProducer(ctx, cfg); err != nil {
		return fmt.Errorf("failed to create kafka producer: %w", err)
	}
	if svc.S3PrivateUploader, svc.S3PublicUploader, err = GetS3Uploaders(cfg); err != nil {
		return fmt.Errorf("failed to initialise s3 uploader: %w", err)
	}
	if svc.S3PrivateDownloader, svc.S3PublicDownloader, err = GetS3Downloaders(cfg); err != nil {
		return fmt.Errorf("failed to initialise s3 uploader: %w", err)
	}
	if !cfg.EncryptionDisabled {
		if svc.VaultClient, err = GetVault(cfg); err != nil {
			return fmt.Errorf("failed to initialise vault client: %w", err)
		}
	}

	// Kafka error logging go-routine
	//	consumer.Channels().LogErrors(ctx, "kafka consumer") !!! does the stuff above have this, or need something like this ?

	svc.DatasetAPIClient = GetDatasetAPIClient(cfg)

	svc.Processor = GetProcessor(cfg)
	svc.generator = GetGenerator()

	// Get HealthCheck
	if svc.HealthCheck, err = GetHealthCheck(cfg, buildTime, gitCommit, version); err != nil {
		return fmt.Errorf("could not instantiate healthcheck: %w", err)
	}

	if err := svc.registerCheckers(); err != nil {
		return fmt.Errorf("error initialising checkers: %w", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true).Path("/health").HandlerFunc(svc.HealthCheck.Handler)
	svc.Server = GetHTTPServer(cfg.BindAddr, r) //!!! what starts the server ?

	return nil
}

// Start the service
func (svc *Service) Start(ctx context.Context, svcErrors chan error) {
	log.Info(ctx, "starting service")

	// Kafka error logging go-routine
	svc.Consumer.Channels().LogErrors(ctx, "kafka consumer")

	// Event Handler for Kafka Consumer
	svc.Processor.Consume(
		ctx,
		svc.Consumer,
		handler.NewXlsxCreate(
			*svc.Cfg,
			svc.DatasetAPIClient,
			svc.S3PrivateUploader,
			svc.S3PublicUploader,
			svc.S3PrivateDownloader,
			svc.S3PublicDownloader,
			svc.VaultClient,
			svc.Producer,
			svc.generator,
		),
	)

	svc.HealthCheck.Start(ctx)

	// Run the http server in a new go-routine
	go func() {
		if err := svc.Server.ListenAndServe(); err != nil {
			svcErrors <- fmt.Errorf("failure in http listen and serve: %w", err)
		}
	}()
}

// Close gracefully shuts the service down in the required order, with timeout
func (svc *Service) Close(ctx context.Context) error {
	timeout := svc.Cfg.GracefulShutdownTimeout
	log.Info(ctx, "commencing graceful shutdown", log.Data{"graceful_shutdown_timeout": timeout})
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	hasShutdownError := false

	go func() {
		defer cancel()

		// stop healthcheck, as it depends on everything else
		if svc.HealthCheck != nil {
			svc.HealthCheck.Stop()
			log.Info(shutdownCtx, "stopped health checker")
		}

		// If kafka consumer exists, stop listening to it.
		// This will automatically stop the event consumer loops and no more messages will be processed.
		// The kafka consumer will be closed after the service shuts down.
		if svc.Consumer != nil {
			if err := svc.Consumer.StopListeningToConsumer(shutdownCtx); err != nil {
				log.Error(shutdownCtx, "error stopping kafka consumer listener", err)
				hasShutdownError = true
			}
			log.Info(shutdownCtx, "stopped kafka consumer listener")
		}

		// stop any incoming requests before closing any outbound connections
		if svc.Server != nil {
			if err := svc.Server.Shutdown(shutdownCtx); err != nil {
				log.Error(shutdownCtx, "failed to shutdown http server", err)
				hasShutdownError = true
			}
			log.Info(shutdownCtx, "stopped http server")
		}

		// If kafka consumer exists, close it.
		if svc.Consumer != nil {
			if err := svc.Consumer.Close(shutdownCtx); err != nil {
				log.Error(shutdownCtx, "error closing kafka consumer", err)
				hasShutdownError = true
			}
			log.Info(shutdownCtx, "closed kafka consumer")
		}
	}()

	// wait for shutdown success (via cancel) or failure (timeout)
	<-shutdownCtx.Done()

	// timeout expired
	if shutdownCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("shutdown timed out: %w", shutdownCtx.Err())
	}

	// other error
	if hasShutdownError {
		return fmt.Errorf("failed to shutdown gracefully")
	}

	log.Info(ctx, "graceful shutdown was successful")
	return nil
}

// registerCheckers adds the checkers for the service clients to the health check object.
func (svc *Service) registerCheckers() error {
	if err := svc.HealthCheck.AddCheck("Kafka consumer", svc.Consumer.Checker); err != nil {
		return fmt.Errorf("error adding check for Kafka consumer: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("Kafka producer", svc.Producer.Checker); err != nil {
		return fmt.Errorf("error adding check for Kafka producer: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("Dataset API client", svc.DatasetAPIClient.Checker); err != nil {
		return fmt.Errorf("error adding check for dataset API client: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("S3 private uploader", svc.S3PrivateUploader.Checker); err != nil {
		return fmt.Errorf("error adding check for s3 private uploader: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("S3 public uploader", svc.S3PublicUploader.Checker); err != nil {
		return fmt.Errorf("error adding check for s3 public uploader: %w", err)
	}

	if !svc.Cfg.EncryptionDisabled {
		if err := svc.HealthCheck.AddCheck("Vault", svc.VaultClient.Checker); err != nil {
			return fmt.Errorf("error adding check for vault client: %w", err)
		}
	}

	return nil
}
