package handler

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ...

// configS3 creates the S3 client
func configS3(your_access_key string, your_secret_key string, aws_s3_bucket string) (*s3.Client, error) {

	creds := credentials.NewStaticCredentialsProvider(your_access_key, your_secret_key, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(creds), config.WithRegion(aws_s3_bucket))
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

// !!! try adjusting code to the v1 lib from:
// https://github.com/antsanchez/goS3example/

// so that it can be more easily adjusted to fit in with rest of ONS code.
