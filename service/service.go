package service

import (
	"context"
	//	"errors"
	"fmt"

	"github.com/pkg/errors"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/log.go/v2/log"

	"github.com/gorilla/mux"
)

// Service contains all the configs, server and clients to run the event handler service
type Service struct {
	Cfg              *config.Config
	Server           HTTPServer
	HealthCheck      HealthChecker
	Consumer         kafka.IConsumerGroup
	DatasetAPIClient DatasetAPIClient
	S3Private        S3Client
	S3Public         S3Client
	VaultClient      VaultClient
	generator        Generator
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
	if svc.S3Private, svc.S3Public, err = GetS3Clients(cfg); err != nil {
		return fmt.Errorf("failed to initialise s3 client: %w", err)
	}
	if svc.VaultClient, err = GetVault(cfg); err != nil {
		return fmt.Errorf("failed to initialise vault client: %w", err)
	}

	svc.DatasetAPIClient = GetDatasetAPIClient(cfg)

	svc.generator = GetGenerator()

	// Event Handler for Kafka Consumer
	h := handler.NewXlsxCreate(
		*svc.Cfg,
		svc.DatasetAPIClient,
		svc.S3Private,
		svc.S3Public,
		svc.VaultClient,
		svc.generator,
	)
	if err := svc.Consumer.RegisterHandler(ctx, h.Handle); err != nil {
		return fmt.Errorf("could not register kafka handler: %w", err)
	}

	// Get HealthCheck
	if svc.HealthCheck, err = GetHealthCheck(cfg, buildTime, gitCommit, version); err != nil {
		return fmt.Errorf("could not instantiate healthcheck: %w", err)
	}

	if err := svc.registerCheckers(); err != nil {
		return fmt.Errorf("error initialising checkers: %w", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(true).Path("/health").HandlerFunc(svc.HealthCheck.Handler)
	svc.Server = GetHTTPServer(cfg.BindAddr, r)

	return nil
}

// Start the service
func (svc *Service) Start(ctx context.Context, svcErrors chan error) error {
	log.Info(ctx, "starting service")

	// Kafka error logging go-routine
	svc.Consumer.LogErrors(ctx)

	// If start/stop on health updates is disabled, start consuming as soon as possible
	if !svc.Cfg.StopConsumingOnUnhealthy {
		if err := svc.Consumer.Start(); err != nil {
			return errors.Wrap(err, "consumer failed to start")
		}
	}

	svc.HealthCheck.Start(ctx)

	// Run the http server in a new go-routine
	go func() {
		if err := svc.Server.ListenAndServe(); err != nil {
			svcErrors <- fmt.Errorf("failure in http listen and serve: %w", err)
		}
	}()

	return nil
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
			if err := svc.Consumer.StopAndWait(); err != nil {
				log.Error(ctx, "failed to stop kafka consumer", err)
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

	if err := svc.HealthCheck.AddCheck("Dataset API client", svc.DatasetAPIClient.Checker); err != nil {
		return fmt.Errorf("error adding check for dataset API client: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("S3 private client", svc.S3Private.Checker); err != nil {
		return fmt.Errorf("error adding check for s3 private client: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("S3 public client", svc.S3Public.Checker); err != nil {
		return fmt.Errorf("error adding check for s3 public client: %w", err)
	}

	if err := svc.HealthCheck.AddCheck("Vault", svc.VaultClient.Checker); err != nil {
		return fmt.Errorf("error adding check for vault client: %w", err)
	}

	if svc.Cfg.StopConsumingOnUnhealthy {
		svc.HealthCheck.SubscribeAll(svc.Consumer)
	}

	return nil
}
