package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/service"
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

	serviceCtx, cancelService := context.WithCancel(ctx)

	// Read config
	cfg, err := config.Get()
	if err != nil {
		cancelService()
		return fmt.Errorf("unable to retrieve service configuration: %w", err)
	}
	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg, "build_time": BuildTime, "git-commit": GitCommit})

	// Run the service
	svc := service.New()
	if err := svc.Init(serviceCtx, cfg, BuildTime, GitCommit, Version); err != nil {
		cancelService()
		return fmt.Errorf("running service failed with error: %w", err)
	}
	if err := svc.Start(ctx, svcErrors); err != nil {
		cancelService()
		return fmt.Errorf("service start failed with error: %w", err)
	}

	// blocks until an os interrupt or a fatal error occurs
	select {
	case err := <-svcErrors:
		err = fmt.Errorf("service error received: %w", err)
		if errClose := svc.Close(serviceCtx); errClose != nil {
			log.Error(serviceCtx, "service close error during error handling", errClose)
		}
		cancelService()
		return err
	case sig := <-signals:
		log.Info(serviceCtx, "os signal received", log.Data{"signal": sig})
	}

	// we do this to cancel 3rd part libraries like AWS S3 access - BEFORE closing our internal code
	cancelService()

	return svc.Close(ctx)
}
