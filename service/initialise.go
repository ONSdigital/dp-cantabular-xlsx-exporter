package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/generator"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dpkafka "github.com/ONSdigital/dp-kafka/v2" //!!! one of these two lines should go
	kafka "github.com/ONSdigital/dp-kafka/v2"
	dphttp "github.com/ONSdigital/dp-net/http"
	dps3 "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const VaultRetries = 3

// GetHTTPServer creates an http server and sets the Server
var GetHTTPServer = func(bindAddr string, router http.Handler) HTTPServer {
	s := dphttp.NewServer(bindAddr, router)
	s.HandleOSSignals = false
	return s
}

// GetKafkaConsumer creates a Kafka consumer
var GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (dpkafka.IConsumerGroup, error) {
	cgChannels := dpkafka.CreateConsumerGroupChannels(1)

	kafkaOffset := dpkafka.OffsetNewest
	if cfg.KafkaOffsetOldest {
		kafkaOffset = dpkafka.OffsetOldest
	}

	return dpkafka.NewConsumerGroup(
		ctx,
		cfg.KafkaAddr,
		cfg.CsvCreatedTopic,
		cfg.CsvCreatedGroup,
		cgChannels,
		&dpkafka.ConsumerGroupConfig{
			KafkaVersion: &cfg.KafkaVersion,
			Offset:       &kafkaOffset,
		},
	)
}

// GetKafkaProducer creates a Kafka producer
var GetKafkaProducer = func(ctx context.Context, cfg *config.Config) (dpkafka.IProducer, error) {
	pChannels := dpkafka.CreateProducerChannels()
	return dpkafka.NewProducer(
		ctx,
		cfg.KafkaAddr,
		cfg.CantabularOutputCreatedTopic,
		pChannels,
		&kafka.ProducerConfig{},
	)
}

// GetS3Uploader creates an S3 Uploader, or a local storage client if a non-empty LocalObjectStore is provided
//!!! should following actually be:
// func (*External) S3Client(cfg *config.Config) (content.S3Client, error) {
// etc as this service will be reading frmo S3 and writting to it ???
var GetS3Uploader = func(cfg *config.Config) (S3Uploader, error) {
	if cfg.LocalObjectStore != "" {
		s3Config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
			Endpoint:         aws.String(cfg.LocalObjectStore),
			Region:           aws.String(cfg.AWSRegion),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}

		s, err := session.NewSession(s3Config)
		if err != nil {
			//!!! should the following actually say (as in doenload-service): "could not create the local-object-store s3 client: %w", err   ???
			return nil, fmt.Errorf("failed to create aws session: %w", err)
		}
		return dps3.NewUploaderWithSession(cfg.UploadBucketName, s), nil
	}

	uploader, err := dps3.NewUploader(cfg.AWSRegion, cfg.UploadBucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 Client: %w", err)
	}

	return uploader, nil
}

// GetVault creates a VaultClient
var GetVault = func(cfg *config.Config) (VaultClient, error) {
	return vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, VaultRetries)
}

// GetProcessor gets and initialises the event Processor
var GetProcessor = func(cfg *config.Config) Processor {
	return event.NewProcessor(*cfg)
}

// GetHealthCheck creates a healthcheck with versionInfo
var GetHealthCheck = func(cfg *config.Config, buildTime, gitCommit, version string) (HealthChecker, error) {
	versionInfo, err := healthcheck.NewVersionInfo(buildTime, gitCommit, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version info: %w", err)
	}

	hc := healthcheck.New(
		versionInfo,
		cfg.HealthCheckCriticalTimeout,
		cfg.HealthCheckInterval,
	)
	return &hc, nil
}

var GetGenerator = func() Generator {
	return generator.New()
}
