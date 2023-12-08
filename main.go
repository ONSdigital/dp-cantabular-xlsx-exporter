package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/service"
	dpotelgo "github.com/ONSdigital/dp-otel-go"
	"github.com/ONSdigital/log.go/v2/log"
)

const serviceName = "dp-cantabular-xlsx-exporter"

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string

	/* NOTE: replace the above with the below to run code with for example vscode debugger.
	BuildTime string = "1601119818"
	GitCommit string = "6584b786caac36b6214ffe04bf62f058d4021538"
	Version   string = "v0.1.0"

	*/
)

func main() {
	log.Namespace = serviceName
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(ctx, "fatal runtime error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)
	svcErrors := make(chan error, 1)

	// Read config
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("unable to retrieve service configuration: %w", err)
	}
	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg, "build_time": BuildTime, "git-commit": GitCommit})

	// Set up OpenTelemetry
	otelConfig := dpotelgo.Config{
		OtelServiceName:          cfg.OTServiceName,
		OtelExporterOtlpEndpoint: cfg.OTExporterOTLPEndpoint,
	}

	otelShutdown, err := dpotelgo.SetupOTelSDK(ctx, otelConfig)

	if err != nil {
		log.Error(ctx, "error setting up OpenTelemetry - hint: ensure OTEL_EXPORTER_OTLP_ENDPOINT is set", err)
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Run the service
	svc := service.New()
	if err := svc.Init(ctx, cfg, BuildTime, GitCommit, Version); err != nil {
		return fmt.Errorf("running service failed with error: %w", err)
	}
	if err := svc.Start(ctx, svcErrors); err != nil {
		return fmt.Errorf("service start failed with error: %w", err)
	}

	// blocks until an os interrupt or a fatal error occurs
	select {
	case err := <-svcErrors:
		err = fmt.Errorf("service error received: %w", err)
		if errClose := svc.Close(ctx); errClose != nil {
			log.Error(ctx, "service close error during error handling", errClose)
		}
		// NOTE: at this point we have tried to close the service, so we should not call cancelService()
		return err

	case sig := <-signals:
		log.Info(ctx, "os signal received", log.Data{"signal": sig})
	}

	return svc.Close(ctx)
}
