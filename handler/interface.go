package handler

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//go:generate moq -out mock/dataset-api-client.go -pkg mock . DatasetAPIClient
//go:generate moq -out mock/s3-client.go -pkg mock . S3Uploader
//go:generate moq -out mock/vault.go -pkg mock . VaultClient
//go:generate moq -out mock/generator.go -pkg mock . Generator

// S3Uploader contains the required method for the S3 Uploader
type S3Uploader interface {
	Upload(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	UploadWithContext(ctx context.Context, input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
	UploadWithPSK(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error)
	BucketName() string
}

// DatasetAPIClient contains the required method for the Dataset API Client
type DatasetAPIClient interface {
	GetInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch string) (i dataset.Instance, eTag string, err error)
	//	PutInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID string, instanceUpdate dataset.UpdateInstance, ifMatch string) (eTag string, err error)
}

// VaultClient contains the required methods for the Vault Client
type VaultClient interface {
	WriteKey(path, key, value string) error
}

// Generator contains methods for dynamically required strings and tokens
// e.g. UUIDs, PSKs.
type Generator interface {
	NewPSK() ([]byte, error)
}
