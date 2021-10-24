package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// KafkaTLSProtocolFlag informs service to use TLS protocol for kafka
const KafkaTLSProtocolFlag = "TLS"

// Config represents service configuration for dp-cantabular-xlsx-exporter
type Config struct {
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	GracefulShutdownTimeout    time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	DownloadServiceURL         string        `envconfig:"DOWNLOAD_SERVICE_URL"` // needed to create url for file downloads, but this service is not actually called - TODO - remove if not needed
	AWSRegion                  string        `envconfig:"AWS_REGION"`
	UploadBucketName           string        `envconfig:"UPLOAD_BUCKET_NAME"`
	LocalObjectStore           string        `envconfig:"LOCAL_OBJECT_STORE"`
	MinioAccessKey             string        `envconfig:"MINIO_ACCESS_KEY"`
	MinioSecretKey             string        `envconfig:"MINIO_SECRET_KEY"`
	OutputFilePath             string        `envconfig:"OUTPUT_FILE_PATH"`
	VaultToken                 string        `envconfig:"VAULT_TOKEN"                   json:"-"`
	VaultAddress               string        `envconfig:"VAULT_ADDR"`
	VaultPath                  string        `envconfig:"VAULT_PATH"`
	ComponentTestUseLogFile    bool          `envconfig:"COMPONENT_TEST_USE_LOG_FILE"` //!!! add to secrets (if needs be)
	EncryptionDisabled         bool          `envconfig:"ENCRYPTION_DISABLED"`
	KafkaConfig                KafkaConfig
}

// KafkaConfig contains the config required to connect to Kafka
type KafkaConfig struct {
	Addr                         []string `envconfig:"KAFKA_ADDR"                            json:"-"`
	Version                      string   `envconfig:"KAFKA_VERSION"`
	OffsetOldest                 bool     `envconfig:"KAFKA_OFFSET_OLDEST"`
	NumWorkers                   int      `envconfig:"KAFKA_NUM_WORKERS"`
	MaxBytes                     int      `envconfig:"KAFKA_MAX_BYTES"`
	SecProtocol                  string   `envconfig:"KAFKA_SEC_PROTO"`
	SecCACerts                   string   `envconfig:"KAFKA_SEC_CA_CERTS"`
	SecClientKey                 string   `envconfig:"KAFKA_SEC_CLIENT_KEY"                  json:"-"`
	SecClientCert                string   `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	SecSkipVerify                bool     `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	CsvCreatedGroup              string   `envconfig:"CSV_CREATED_GROUP"`               // this is the consumed group, and is only defined for the consumer(s)
	CsvCreatedTopic              string   `envconfig:"CSV_CREATED_TOPIC"`               // this is the consumed topic
	CantabularOutputCreatedTopic string   `envconfig:"CANTABULAR_OUTPUT_CREATED_TOPIC"` // this is produced
}

var cfg *Config

// Get returns the default config with any modifications through environment
// variables
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                   "localhost:26800",
		GracefulShutdownTimeout:    5 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		DownloadServiceURL:         "http://localhost:23600",
		AWSRegion:                  "eu-west-1",
		UploadBucketName:           "dp-cantabular-csv-exporter", // needed for place to download .csv from
		LocalObjectStore:           "",
		MinioAccessKey:             "",                    // in develop & prod this is also the AWS_ACCESS_KEY_ID
		MinioSecretKey:             "",                    // in develop & prod this is also the AWS_SECRET_ACCESS_KEY
		OutputFilePath:             "/tmp/helloworld.txt", // TODO remove this
		VaultPath:                  "secret/shared/psk",   // TODO - remove if not needed
		VaultAddress:               "http://localhost:8200",
		VaultToken:                 "",
		ComponentTestUseLogFile:    false,
		EncryptionDisabled:         false, // needed for local development to skip needing vault - TODO - remove if not needed
		KafkaConfig: KafkaConfig{
			Addr:                         []string{"localhost:9092"},
			Version:                      "1.0.2",
			OffsetOldest:                 true,
			NumWorkers:                   1, // it is not advised to change this, as it may cause Out of Memeory problems for very large files - TBD
			MaxBytes:                     2000000,
			SecProtocol:                  "",
			SecCACerts:                   "",
			SecClientKey:                 "",
			SecClientCert:                "",
			SecSkipVerify:                false,
			CsvCreatedGroup:              "dp-cantabular-xlsx-exporter",
			CsvCreatedTopic:              "cantabular-csv-created",
			CantabularOutputCreatedTopic: "cantabular-output-created",
		},
	}

	return cfg, envconfig.Process("", cfg)
}
