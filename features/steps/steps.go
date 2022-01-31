package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/cucumber/godog"
)

// RegisterSteps maps the human-readable regular expressions to their corresponding functions
func (c *Component) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the following instance with id "([^"]*)" is available from dp-dataset-api:$`, c.theFollowingInstanceIsAvailable)
	ctx.Step(`^an instance with id "([^"]*)" is updated to dp-dataset-api`, c.theFollowingInstanceIsUpdated)
	ctx.Step(`^the service starts`, c.theServiceStarts)
	ctx.Step(`^dp-dataset-api is healthy`, c.datasetAPIIsHealthy)
	ctx.Step(`^dp-dataset-api is unhealthy`, c.datasetAPIIsUnhealthy)
	ctx.Step(`^this cantabular-csv-created event is queued, to be consumed:$`, c.thisCantabularCsvCreatedEventIsQueued)
	ctx.Step(`^a file with filename "([^"]*)" can be seen in minio`, c.theFollowingFileCanBeSeenInMinio)
	//	ctx.Step(`^no file with filename "([^"]*)" can be seen in minio`, c.theFollowingFileCannotBeSeenInMinio) !!! is this function needed ?
}

// theServiceStarts starts the service under test in a new go-routine
// note that this step should be called only after all dependencies have been set up,
// to prevent any race condition, specially during the first healthcheck iteration.
func (c *Component) theServiceStarts() error {
	return c.startService(c.ctx)
}

// datasetAPIIsHealthy generates a mocked healthy response for dataset API healthcheck
func (c *Component) datasetAPIIsHealthy() error {
	const res = `{"status": "OK"}`
	c.DatasetAPI.NewHandler().
		Get("/health").
		Reply(http.StatusOK).
		BodyString(res)
	return nil
}

// datasetAPIIsUnhealthy generates a mocked unhealthy response for dataset API healthcheck
func (c *Component) datasetAPIIsUnhealthy() error {
	const res = `{"status": "CRITICAL"}`
	c.DatasetAPI.NewHandler().
		Get("/health").
		Reply(http.StatusInternalServerError).
		BodyString(res)
	return nil
}

// theFollowingInstanceIsAvailable generate a mocked response for dataset API
// GET /instances/{id} with the provided instance response
func (c *Component) theFollowingInstanceIsAvailable(id string, instance *godog.DocString) error {
	c.DatasetAPI.NewHandler().
		Get("/instances/"+id).
		Reply(http.StatusOK).
		BodyString(instance.Content).
		AddHeader("Etag", c.testETag)

	return nil
}

// theFollowingInstanceIsUpdated generate a mocked response for dataset API
// PUT /instances/{id} with the provided instance response
func (c *Component) theFollowingInstanceIsUpdated(id string) error {
	c.DatasetAPI.NewHandler().
		Put("/instances/"+id).
		Reply(http.StatusOK).
		AddHeader("Etag", c.testETag)

	return nil
}

// thisCantabularCsvCreatedEventIsQueued produces a new ExportStart event with the contents defined by the input
func (c *Component) thisCantabularCsvCreatedEventIsQueued(input *godog.DocString) error {
	var testEvent event.CantabularCsvCreated
	if err := json.Unmarshal([]byte(input.Content), &testEvent); err != nil {
		return fmt.Errorf("error unmarshaling input to event: %w body: %s", err, input.Content)
	}

	log.Info(c.ctx, "event to send for testing: ", log.Data{
		"event": testEvent,
	})

	if err := c.producer.Send(schema.CantabularCsvCreated, testEvent); err != nil {
		return fmt.Errorf("failed to send event for testing: %w", err)
	}
	return nil
}

// theFollowingPublicFileCanBeSeenInMinio checks that the provided fileName is available in the public bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingPublicFileCanBeSeenInMinio(fileName string) error {
	return c.theFollowingFileCanBeSeenInMinio(fileName, c.cfg.PublicBucketName)
}

// theFollowingPrivateFileCanBeSeenInMinio checks that the provided fileName is available in the private bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingPrivateFileCanBeSeenInMinio(fileName string) error {
	return c.theFollowingFileCanBeSeenInMinio(fileName, c.cfg.PrivateBucketName)
}

// theFollowingFileCanBeSeenInMinio checks that the provided fileName is available in the provided bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingFileCanBeSeenInMinio(fileName, bucketName string) error {
	var b []byte
	f := aws.NewWriteAtBuffer(b)

	// probe bucket with backoff to give time for event to be processed
	retries := MinioCheckRetries
	timeout := time.Second
	var numBytes int64
	var err error

	for {
		if numBytes, err = c.S3Downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
		}); err == nil || retries <= 0 {
			break
		}

		retries--

		log.Info(c.ctx, "error obtaining file from minio. Retrying.", log.Data{
			"error":        err,
			"retries_left": retries,
		})

		time.Sleep(timeout)
		timeout *= 2
	}
	if err != nil {
		return fmt.Errorf(
			"error obtaining file from minio. Last error: %w",
			err,
		)
	}

	if numBytes < 1 {
		return errors.New("file length zero")
	}

	log.Info(c.ctx, "got file contents", log.Data{
		"contents": string(f.Bytes()),
	})

	return nil
}
