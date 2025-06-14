package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/population"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/generator"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	kafka "github.com/ONSdigital/dp-kafka/v4"
	dphttp "github.com/ONSdigital/dp-net/http"
	dps3 "github.com/ONSdigital/dp-s3/v3"
	vault "github.com/ONSdigital/dp-vault"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const VaultRetries = 3

var OneAndOnlyOneWorker = 1 // WARNING - Do NOT EVER make this bigger than '1' otherwise an OOM might happen for more than one large csv file being processed in parallel

// GetHTTPServer creates a http server and sets the Server
var GetHTTPServer = func(bindAddr string, router http.Handler) HTTPServer {
	otelHandler := otelhttp.NewHandler(router, "/")
	s := dphttp.NewServer(bindAddr, otelHandler)
	s.HandleOSSignals = false
	return s
}

// GetCantabularClient gets and initialises the Cantabular Client
var GetCantabularClient = func(cfg *config.Config) CantabularClient {
	return cantabular.NewClient(
		cantabular.Config{
			Host:           cfg.CantabularURL,
			ExtApiHost:     cfg.CantabularExtURL,
			GraphQLTimeout: cfg.DefaultRequestTimeout,
		},
		dphttp.NewClient(),
		nil,
	)
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

// GetPopulationTypesAPIClient gets and initialises the PopulationTypesAPI Client
var GetPopulationTypesAPIClient = func(cfg *config.Config) (PopulationTypesAPIClient, error) {
	return population.NewClient(cfg.PopulationTypesAPIURL)
}

// GetS3Clients creates the private and public S3 Clients using the same AWS config, or a local storage client if a non-empty LocalObjectStore is provided
var GetS3Clients = func(ctx context.Context, cfg *config.Config) (privateClient, publicClient S3Client, err error) {
	if cfg.LocalObjectStore != "" {
		awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
			awsConfig.WithRegion(cfg.AWSRegion),
			awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.MinioAccessKey, cfg.MinioSecretKey, "")),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create aws config (local): %w", err)
		}

		privateClient = dps3.NewClientWithConfig(cfg.PrivateBucketName, awsConfig, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.LocalObjectStore)
			o.UsePathStyle = true
		})

		publicClient = dps3.NewClientWithConfig(cfg.PublicBucketName, awsConfig, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.LocalObjectStore)
			o.UsePathStyle = true
		})
		return privateClient, publicClient, nil
	}

	awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create aws config: %w", err)
	}

	privateClient = dps3.NewClientWithConfig(cfg.PrivateBucketName, awsConfig)
	publicClient = dps3.NewClientWithConfig(cfg.PublicBucketName, awsConfig)

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
