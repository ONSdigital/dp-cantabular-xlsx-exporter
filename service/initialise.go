package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/generator"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	kafka "github.com/ONSdigital/dp-kafka/v3"
	dphttp "github.com/ONSdigital/dp-net/http"
	dps3 "github.com/ONSdigital/dp-s3/v2"
	vault "github.com/ONSdigital/dp-vault"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

const VaultRetries = 3

var OneAndOnlyOneWorker = 1 // WARNING - Do NOT EVER make this bigger than '1' otherwise an OOM might happen for more than one large csv file being processed in parallel

// GetHTTPServer creates a http server and sets the Server
var GetHTTPServer = func(bindAddr string, router http.Handler) HTTPServer {
	s := dphttp.NewServer(bindAddr, router)
	s.HandleOSSignals = false
	return s
}

// GetKafkaConsumer creates a Kafka consumer
var GetKafkaConsumer = func(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error) {
	kafkaOffset := kafka.OffsetNewest
	if cfg.KafkaConfig.OffsetOldest {
		kafkaOffset = kafka.OffsetOldest
	}
	cgConfig := &kafka.ConsumerGroupConfig{
		BrokerAddrs:       cfg.KafkaConfig.Addr,
		Topic:             cfg.KafkaConfig.CsvCreatedTopic,
		GroupName:         cfg.KafkaConfig.CsvCreatedGroup,
		MinBrokersHealthy: &cfg.KafkaConfig.ConsumerMinBrokersHealthy,
		KafkaVersion:      &cfg.KafkaConfig.Version,
		NumWorkers:        &OneAndOnlyOneWorker,
		Offset:            &kafkaOffset,
	}
	if cfg.KafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		cgConfig.SecurityConfig = kafka.GetSecurityConfig(
			cfg.KafkaConfig.SecCACerts,
			cfg.KafkaConfig.SecClientCert,
			cfg.KafkaConfig.SecClientKey,
			cfg.KafkaConfig.SecSkipVerify,
		)
	}
	return kafka.NewConsumerGroup(ctx, cgConfig)
}

// GetDatasetAPIClient gets and initialises the DatasetAPI Client
var GetDatasetAPIClient = func(cfg *config.Config) DatasetAPIClient {
	return dataset.NewAPIClient(cfg.DatasetAPIURL)
}

// GetFilterAPIClient gets and initialises the FilterAPI Client
var GetFilterAPIClient = func(cfg *config.Config) FilterAPIClient {
	return filter.New(cfg.FilterAPIURL)
}

// GetS3Clients creates the private and public S3 Clients using the same AWS session, or a local storage client if a non-empty LocalObjectStore is provided
var GetS3Clients = func(cfg *config.Config) (privateClient, publicClient S3Client, err error) {
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
		return dps3.NewClientWithSession(cfg.PrivateBucketName, s),
			dps3.NewClientWithSession(cfg.PublicBucketName, s),
			nil
	}

	privateClient, err = dps3.NewClient(cfg.AWSRegion, cfg.PrivateBucketName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create S3 Client: %w", err)
	}

	publicClient = dps3.NewClientWithSession(cfg.PublicBucketName, privateClient.Session())
	return privateClient, publicClient, nil
}

// GetVault creates a VaultClient
var GetVault = func(cfg *config.Config) (VaultClient, error) {
	return vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, VaultRetries)
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
