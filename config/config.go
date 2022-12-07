package config

import (
	"time"

	mongo "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	"github.com/kelseyhightower/envconfig"
)

// KafkaTLSProtocolFlag informs service to use TLS protocol for kafka
const KafkaTLSProtocolFlag = "TLS"

// Config represents service configuration for dp-cantabular-xlsx-exporter
type Config struct {
	// in develop & prod there is also AWS_ACCESS_KEY_ID & AWS_SECRET_ACCESS_KEY in the secrets files
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	GracefulShutdownTimeout    time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	ServiceAuthToken           string        `envconfig:"SERVICE_AUTH_TOKEN"         json:"-"`
	DatasetAPIURL              string        `envconfig:"DATASET_API_URL"`
	FilterAPIURL               string        `envconfig:"FILTER_API_URL"`
	PopulationTypesAPIURL      string        `envconfig:"POPULATION_TYPES_API_URL"`
	FiltersCollection          string        `envconfig:"FILTERS_COLLECTION"`
	FilterOutputsCollection    string        `envconfig:"FILTER_OUTPUTS_COLLECTION"`
	DownloadServiceURL         string        `envconfig:"DOWNLOAD_SERVICE_URL"` // needed to create url for file downloads
	AWSRegion                  string        `envconfig:"AWS_REGION"`
	PublicBucketName           string        `envconfig:"UPLOAD_BUCKET_NAME"`
	PrivateBucketName          string        `envconfig:"PRIVATE_UPLOAD_BUCKET_NAME"`
	LocalObjectStore           string        `envconfig:"LOCAL_OBJECT_STORE"`
	MinioAccessKey             string        `envconfig:"MINIO_ACCESS_KEY"`
	MinioSecretKey             string        `envconfig:"MINIO_SECRET_KEY"`
	VaultToken                 string        `envconfig:"VAULT_TOKEN"                   json:"-"`
	VaultAddress               string        `envconfig:"VAULT_ADDR"`
	VaultPath                  string        `envconfig:"VAULT_PATH"`
	ComponentTestUseLogFile    bool          `envconfig:"COMPONENT_TEST_USE_LOG_FILE"`
	StopConsumingOnUnhealthy   bool          `envconfig:"STOP_CONSUMING_ON_UNHEALTHY"`
	S3PublicURL                string        `envconfig:"S3_PUBLIC_URL"`
	KafkaConfig                KafkaConfig
	Mongo                      mongo.MongoDriverConfig
}

// KafkaConfig contains the config required to connect to Kafka
type KafkaConfig struct {
	Addr                         []string `envconfig:"KAFKA_ADDR"                            json:"-"`
	ConsumerMinBrokersHealthy    int      `envconfig:"KAFKA_CONSUMER_MIN_BROKERS_HEALTHY"`
	ProducerMinBrokersHealthy    int      `envconfig:"KAFKA_PRODUCER_MIN_BROKERS_HEALTHY"`
	Version                      string   `envconfig:"KAFKA_VERSION"`
	OffsetOldest                 bool     `envconfig:"KAFKA_OFFSET_OLDEST"`
	MaxBytes                     int      `envconfig:"KAFKA_MAX_BYTES"`
	SecProtocol                  string   `envconfig:"KAFKA_SEC_PROTO"`
	SecCACerts                   string   `envconfig:"KAFKA_SEC_CA_CERTS"`
	SecClientKey                 string   `envconfig:"KAFKA_SEC_CLIENT_KEY"                  json:"-"`
	SecClientCert                string   `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	SecSkipVerify                bool     `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	CsvCreatedGroup              string   `envconfig:"CSV_CREATED_GROUP"`               // this is the consumed group, and is only defined for the consumer(s)
	CsvCreatedTopic              string   `envconfig:"CSV_CREATED_TOPIC"`               // this is the consumed topic
	CantabularOutputCreatedTopic string   `envconfig:"CANTABULAR_OUTPUT_CREATED_TOPIC"` // this is produced ... This may get used if we have an export manager service - TBD
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
		GracefulShutdownTimeout:    25 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		ServiceAuthToken:           "",
		DatasetAPIURL:              "http://localhost:22000",
		FilterAPIURL:               "http://localhost:22100",
		PopulationTypesAPIURL:      "http://localhost:27300",
		FiltersCollection:          "filters",
		FilterOutputsCollection:    "filterOutputs",
		DownloadServiceURL:         "http://localhost:23600",
		AWSRegion:                  "eu-west-1",
		PublicBucketName:           "public-bucket", // where to place the created .xlsx
		PrivateBucketName:          "private-bucket",
		LocalObjectStore:           "",
		MinioAccessKey:             "",
		MinioSecretKey:             "",
		VaultPath:                  "secret/shared/psk",
		VaultAddress:               "http://localhost:8200",
		VaultToken:                 "",
		ComponentTestUseLogFile:    false,
		StopConsumingOnUnhealthy:   true,
		S3PublicURL:                "http://public-bucket",
		KafkaConfig: KafkaConfig{
			Addr:                         []string{"localhost:9092", "localhost:9093", "localhost:9094"},
			ConsumerMinBrokersHealthy:    1,
			ProducerMinBrokersHealthy:    2,
			Version:                      "1.0.2",
			OffsetOldest:                 true,
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
		Mongo: mongo.MongoDriverConfig{
			ClusterEndpoint: "localhost:27017",
			Username:        "",
			Password:        "",
			Database:        "filters",
			Collections: map[string]string{
				"filters":       "filters",
				"filterOutputs": "filterOutputs",
			},
			ReplicaSet:                    "",
			IsStrongReadConcernEnabled:    false,
			IsWriteConcernMajorityEnabled: true,
			ConnectTimeout:                5 * time.Second,
			QueryTimeout:                  15 * time.Second,
			TLSConnectionConfig: mongo.TLSConnectionConfig{
				IsSSL: false,
			},
		},
	}

	return cfg, envconfig.Process("", cfg)
}
