package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/generator"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	kafka "github.com/ONSdigital/dp-kafka/v2"
	dphttp "github.com/ONSdigital/dp-net/http"
	dps3 "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const VaultRetries = 3

// GetHTTPServer creates a http server and sets the Server
var GetHTTPServer = func(bindAddr string, router http.Handler) HTTPServer {
	s := dphttp.NewServer(bindAddr, router)
	s.HandleOSSignals = false
	return s
}

// GetKafkaConsumer creates a Kafka consumer
var GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
	cgChannels := kafka.CreateConsumerGroupChannels(cfg.KafkaConfig.NumWorkers)

	kafkaOffset := kafka.OffsetNewest
	if cfg.KafkaConfig.OffsetOldest {
		kafkaOffset = kafka.OffsetOldest
	}
	cgConfig := &kafka.ConsumerGroupConfig{
		KafkaVersion: &cfg.KafkaConfig.Version,
		Offset:       &kafkaOffset,
	}
	if cfg.KafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		cgConfig.SecurityConfig = kafka.GetSecurityConfig(
			cfg.KafkaConfig.SecCACerts,
			cfg.KafkaConfig.SecClientCert,
			cfg.KafkaConfig.SecClientKey,
			cfg.KafkaConfig.SecSkipVerify,
		)
	}
	return kafka.NewConsumerGroup(
		ctx,
		cfg.KafkaConfig.Addr,
		cfg.KafkaConfig.CsvCreatedTopic,
		cfg.KafkaConfig.CsvCreatedGroup,
		cgChannels,
		cgConfig,
	)
}

// GetKafkaProducer creates a Kafka producer
var GetKafkaProducer = func(ctx context.Context, cfg *config.Config) (kafka.IProducer, error) {
	pChannels := kafka.CreateProducerChannels()
	pConfig := &kafka.ProducerConfig{
		KafkaVersion:    &cfg.KafkaConfig.Version,
		MaxMessageBytes: &cfg.KafkaConfig.MaxBytes,
	}
	if cfg.KafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		pConfig.SecurityConfig = kafka.GetSecurityConfig(
			cfg.KafkaConfig.SecCACerts,
			cfg.KafkaConfig.SecClientCert,
			cfg.KafkaConfig.SecClientKey,
			cfg.KafkaConfig.SecSkipVerify,
		)
	}
	return kafka.NewProducer(
		ctx,
		cfg.KafkaConfig.Addr,
		cfg.KafkaConfig.CantabularOutputCreatedTopic,
		pChannels,
		pConfig,
	)
}

// GetDatasetAPIClient gets and initialises the DatasetAPI Client
var GetDatasetAPIClient = func(cfg *config.Config) DatasetAPIClient {
	return dataset.NewAPIClient(cfg.DatasetAPIURL)
}

// GetS3Uploaders creates the private and public S3 Uploaders using the same AWS session, or a local storage client if a non-empty LocalObjectStore is provided
var GetS3Uploaders = func(cfg *config.Config) (privateUploader, publicUploader S3Client, err error) {
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
			return nil, nil, fmt.Errorf("failed to create aws session (local): %w", err)
		}
		return dps3.NewUploaderWithSession(cfg.PrivateBucketName, s),
			dps3.NewUploaderWithSession(cfg.PublicBucketName, s),
			nil
	}

	privateUploader, err = dps3.NewUploader(cfg.AWSRegion, cfg.PrivateBucketName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create S3 Client: %w", err)
	}

	publicUploader = dps3.NewUploaderWithSession(cfg.PublicBucketName, privateUploader.Session())
	return privateUploader, publicUploader, nil
}

// GetS3Downloaders creates the private and public S3 Downloaders using the same AWS session, or a local storage client if a non-empty LocalObjectStore is provided
var GetS3Downloaders = func(cfg *config.Config) (privateDownloader, publicDownloader *dps3.S3, err error) {
	if cfg.LocalObjectStore != "" {
		// configure things for development utilising minio
		s3Config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
			Endpoint:         aws.String(cfg.LocalObjectStore),
			Region:           aws.String(cfg.AWSRegion),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}

		sess, err := session.NewSession(s3Config)
		if err != nil {
			return nil, nil, fmt.Errorf("could not create the local-object-store s3 client: %w", err)
		}
		return dps3.NewClientWithSession(cfg.PrivateBucketName, sess), dps3.NewClientWithSession(cfg.PublicBucketName, sess), nil
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(cfg.AWSRegion)})
	if err != nil {
		return nil, nil, fmt.Errorf("could not create the s3 client: %w", err)
	}

	return dps3.NewClientWithSession(cfg.PrivateBucketName, sess), dps3.NewClientWithSession(cfg.PublicBucketName, sess), nil
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
