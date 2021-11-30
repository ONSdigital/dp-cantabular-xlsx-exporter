package handler

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//go:generate moq -out mock/dataset-api-client.go -pkg mock . DatasetAPIClient
//go:generate moq -out mock/s3-client.go -pkg mock . S3Uploader
//go:generate moq -out mock/vault.go -pkg mock . VaultClient
//go:generate moq -out mock/generator.go -pkg mock . Generator

// S3Uploader contains the required method for the S3 Uploader
type S3Uploader interface {
	Head(key string) (*s3.HeadObjectOutput, error)
	UploadWithContext(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	UploadWithPSKAndContext(ctx context.Context, input *s3manager.UploadInput, psk []byte, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	BucketName() string
}

// DatasetAPIClient contains the required method for the Dataset API Client
type DatasetAPIClient interface {
	PutVersion(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, datasetID, edition, version string, m dataset.Version) error
	GetInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch string) (i dataset.Instance, eTag string, err error)
	GetVersionMetadata(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, id, edition, version string) (dataset.Metadata, error)
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
