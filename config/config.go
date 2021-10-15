package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// TODO: remove hello call config options
// Config represents service configuration for dp-cantabular-xlsx-exporter
type Config struct {
	BindAddr                     string        `envconfig:"BIND_ADDR"`
	GracefulShutdownTimeout      time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval          time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout   time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	KafkaAddr                    []string      `envconfig:"KAFKA_ADDR"                     json:"-"`
	KafkaVersion                 string        `envconfig:"KAFKA_VERSION"`
	KafkaOffsetOldest            bool          `envconfig:"KAFKA_OFFSET_OLDEST"`
	KafkaNumWorkers              int           `envconfig:"KAFKA_NUM_WORKERS"`
	CsvCreatedGroup              string        `envconfig:"CSV_CREATED_GROUP"`               // this is the consumed group, and is only defined for the consumer(s)
	CsvCreatedTopic              string        `envconfig:"CSV_CREATED_TOPIC"`               // this is the consumed topic
	CantabularOutputCreatedTopic string        `envconfig:"CANTABULAR_OUTPUT_CREATED_TOPIC"` // this is produced
	EncryptionDisabled           bool          `envconfig:"ENCRYPTION_DISABLED"`
	VaultToken                   string        `envconfig:"VAULT_TOKEN"                   json:"-"`
	VaultAddress                 string        `envconfig:"VAULT_ADDR"`
	VaultPath                    string        `envconfig:"VAULT_PATH"`
	DownloadServiceURL           string        `envconfig:"DOWNLOAD_SERVICE_URL"` // needed to create url for file downloads, but this service is not actually called - TODO - remove if not needed
	AWSRegion                    string        `envconfig:"AWS_REGION"`
	UploadBucketName             string        `envconfig:"UPLOAD_BUCKET_NAME"`
	MinioAccessKey               string        `envconfig:"MINIO_ACCESS_KEY"`
	MinioSecretKey               string        `envconfig:"MINIO_SECRET_KEY"`
	OutputFilePath               string        `envconfig:"OUTPUT_FILE_PATH"`
}

var cfg *Config

// Get returns the default config with any modifications through environment
// variables
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                     "localhost:26800",
		GracefulShutdownTimeout:      5 * time.Second,
		HealthCheckInterval:          30 * time.Second,
		HealthCheckCriticalTimeout:   90 * time.Second,
		KafkaAddr:                    []string{"localhost:9092"},
		KafkaVersion:                 "1.0.2",
		KafkaOffsetOldest:            true,
		KafkaNumWorkers:              1,
		CsvCreatedGroup:              "dp-cantabular-xlsx-exporter",
		CsvCreatedTopic:              "cantabular-csv-created",
		CantabularOutputCreatedTopic: "cantabular-output-created",
		EncryptionDisabled:           false,               // needed for local development to skip needing vault - TODO - remove if not needed
		VaultPath:                    "secret/shared/psk", // TODO - remove if not needed
		VaultAddress:                 "http://localhost:8200",
		VaultToken:                   "",
		DownloadServiceURL:           "http://localhost:23600",
		AWSRegion:                    "eu-west-1",
		UploadBucketName:             "dp-cantabular-csv-exporter", // needed for place to download .csv from
		MinioAccessKey:               "",                           // in develop & prod this is also the AWS_ACCESS_KEY_ID
		MinioSecretKey:               "",                           // in develop & prod this is also the AWS_SECRET_ACCESS_KEY
		OutputFilePath:               "/tmp/helloworld.txt",        // TODO remove this
	}

	return cfg, envconfig.Process("", cfg)
}
