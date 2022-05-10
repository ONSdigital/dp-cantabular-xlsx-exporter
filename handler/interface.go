package handler

import (
	"context"
	"io"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//go:generate moq -out mock/dataset-api-client.go -pkg mock . DatasetAPIClient
//go:generate moq -out mock/s3-client.go -pkg mock . S3Client
//go:generate moq -out mock/vault.go -pkg mock . VaultClient
//go:generate moq -out mock/generator.go -pkg mock . Generator

// S3Client contains the required method for the S3 Client
type S3Client interface {
	Get(key string) (io.ReadCloser, *int64, error)
	GetWithPSK(key string, psk []byte) (io.ReadCloser, *int64, error)
	Head(key string) (*s3.HeadObjectOutput, error)
	UploadWithContext(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	UploadWithPSKAndContext(ctx context.Context, input *s3manager.UploadInput, psk []byte, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	BucketName() string
}

// DatasetAPIClient contains the required method for the Dataset API Client
type DatasetAPIClient interface {
	PutVersion(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, datasetID, edition, version string, m dataset.Version) error
	GetInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch string) (i dataset.Instance, eTag string, err error)
	GetVersionMetadataSelection(context.Context, dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error)
}

// VaultClient contains the required methods for the Vault Client
type VaultClient interface {
	ReadKey(path, key string) (string, error)
	WriteKey(path, key, value string) error
}

// Generator contains methods for dynamically required strings and tokens
// e.g. UUIDs, PSKs.
type Generator interface {
	NewPSK() ([]byte, error)
}
