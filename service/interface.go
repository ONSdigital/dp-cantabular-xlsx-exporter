package service

import (
	"context"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-api-clients-go/v2/cantabular"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/population"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	kafka "github.com/ONSdigital/dp-kafka/v4"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:generate moq -out mock/server.go -pkg mock . HTTPServer
//go:generate moq -out mock/health_check.go -pkg mock . HealthChecker
//go:generate moq -out mock/dataset_api_client.go -pkg mock . DatasetAPIClient
//go:generate moq -out mock/s3_client.go -pkg mock . S3Client
//go:generate moq -out mock/vault.go -pkg mock . VaultClient

// Initialiser defines the methods to initialise external services
type Initialiser interface {
	DoGetHTTPServer(bindAddr string, router http.Handler) HTTPServer
	DoGetHealthCheck(cfg *config.Config, buildTime, gitCommit, version string) (HealthChecker, error)
	DoGetKafkaConsumer(ctx context.Context, cfg *config.Config) (kafka.IConsumerGroup, error)
}

// HTTPServer defines the required methods from the HTTP server
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// HealthChecker defines the required methods from Healthcheck
type HealthChecker interface {
	Handler(w http.ResponseWriter, req *http.Request)
	Start(ctx context.Context)
	Stop()
	AddCheck(name string, checker healthcheck.Checker) (err error)
	SubscribeAll(s healthcheck.Subscriber)
}

type CantabularClient interface {
	GetDimensionsByName(context.Context, cantabular.GetDimensionsByNameRequest) (*cantabular.GetDimensionsResponse, error)
}

type DatasetAPIClient interface {
	PutVersion(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, datasetID, edition, version string, m dataset.Version) error
	GetInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch string) (m dataset.Instance, eTag string, err error)
	Checker(context.Context, *healthcheck.CheckState) error
	GetVersionMetadataSelection(context.Context, dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error)
	GetVersions(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceAuthToken, collectionID, datasetID, edition string, q *dataset.QueryParams) (dataset.VersionsList, error)
}
type S3Client interface {
	Get(ctx context.Context, key string) (io.ReadCloser, *int64, error)
	GetWithPSK(ctx context.Context, key string, psk []byte) (io.ReadCloser, *int64, error)
	Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error)
	Upload(ctx context.Context, input *s3.PutObjectInput, options ...func(*manager.Uploader)) (*manager.UploadOutput, error)
	UploadWithPSK(ctx context.Context, input *s3.PutObjectInput, psk []byte) (*manager.UploadOutput, error)
	BucketName() string
	Config() aws.Config
	Checker(context.Context, *healthcheck.CheckState) error
}

type VaultClient interface {
	ReadKey(path, key string) (string, error)
	WriteKey(path, key, value string) error
	Checker(context.Context, *healthcheck.CheckState) error
}

type FilterAPIClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutput string) (m filter.Model, err error)
	UpdateFilterOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, filterOutputID string, m *filter.Model) error
}

type PopulationTypesAPIClient interface {
	GetAreaTypes(ctx context.Context, input population.GetAreaTypesInput) (population.GetAreaTypesResponse, error)
}

// Generator contains methods for dynamically required strings and tokens
// e.g. UUIDs, PSKs.
type Generator interface {
	NewPSK() ([]byte, error)
}
