package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gedge/go-spew/spew"

	"github.com/cucumber/godog"
)

// RegisterSteps maps the human-readable regular expressions to their corresponding functions
func (c *Component) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(
		`^the following metadata document with dataset id "([^"]*)", edition "([^"]*)" and version "([^"]*)" is available from dp-dataset-api:$`,
		c.theFollowingMetadataDocumentIsAvailable,
	)
	ctx.Step(
		`^the following version document with dataset id "([^"]*)", edition "([^"]*)" and version "([^"]*)" is available from dp-dataset-api:$`,
		c.theFollowingVersionDocumentIsAvailable,
	)

	ctx.Step(`^the following Csv file named: "([^"]*)" is available in Public S3 bucket:$`, c.thisFileIsPutInPublicS3Bucket)
	ctx.Step(`^the following instance with id "([^"]*)" is available from dp-dataset-api:$`, c.theFollowingInstanceIsAvailable)
	//	ctx.Step(`^an instance with id "([^"]*)" is updated to dp-dataset-api`, c.theFollowingInstanceIsUpdated)
	ctx.Step(`^the service starts`, c.theServiceStarts)
	ctx.Step(`^dp-dataset-api is healthy`, c.datasetAPIIsHealthy)
	ctx.Step(`^dp-dataset-api is unhealthy`, c.datasetAPIIsUnhealthy)
	ctx.Step(`^a dataset version with dataset-id "([^"]*)", edition "([^"]*)" and version "([^"]*)" is updated by an API call to dp-dataset-api`, c.theFollowingVersionIsUpdated)
	ctx.Step(`^this cantabular-csv-created event is queued, to be consumed:$`, c.thisCantabularCsvCreatedEventIsQueued)
	ctx.Step(`^a public file with filename "([^"]*)" can be seen in minio`, c.theFollowingPublicFileCanBeSeenInMinio)
	ctx.Step(`^a private file with filename "([^"]*)" can be seen in minio`, c.theFollowingPrivateFileCanBeSeenInMinio)
	ctx.Step(`^no public file with filename "([^"]*)" can be seen in minio`, c.theFollowingPublicFileCannotBeSeenInMinio)
	ctx.Step(`^no private file with filename "([^"]*)" can be seen in minio`, c.theFollowingPrivateFileCannotBeSeenInMinio)
}

// theServiceStarts starts the service under test in a new go-routine
// note that this step should be called only after all dependencies have been set up,
// to prevent any race condition, specially during the first healthcheck iteration.
func (c *Component) theServiceStarts() error {
	return c.startService(c.ctx)
}

// theFollowingMetadataDocumentIsAvailable generate a mocked response for dataset API
// GET /datasets/{dataset_id}/editions/{edition}/versions/{version}/metadata
func (c *Component) theFollowingMetadataDocumentIsAvailable(datasetID, edition, version string, md *godog.DocString) error {
	url := fmt.Sprintf(
		"/datasets/%s/editions/%s/versions/%s/metadata",
		datasetID,
		edition,
		version,
	)

	c.DatasetAPI.NewHandler().
		Get(url).
		Reply(http.StatusOK).
		BodyString(md.Content)

	return nil
}

// theFollowingVersionDocumentIsAvailable generates a mocked response for dataset API
// GET /datasets/{dataset_id}/editions/{edition}/versions/{version}
func (c *Component) theFollowingVersionDocumentIsAvailable(datasetID, edition, version string, v *godog.DocString) error {
	url := fmt.Sprintf(
		"/datasets/%s/editions/%s/versions/%s",
		datasetID,
		edition,
		version,
	)

	c.DatasetAPI.NewHandler().
		Get(url).
		Reply(http.StatusOK).
		BodyString(v.Content)

	return nil
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
/*func (c *Component) theFollowingInstanceIsUpdated(id string) error {
	c.DatasetAPI.NewHandler().
		Put("/instances/"+id).
		Reply(http.StatusOK).
		AddHeader("Etag", c.testETag)

	return nil
}*/

// theFollowingVersionIsUpdated generate a mocked response for dataset API
// PUT /datasets/{dataset_id}/editions/{edition}/versions/{version}
func (c *Component) theFollowingVersionIsUpdated(datasetID, edition, version string) error {
	c.DatasetAPI.NewHandler().
		Put("/datasets/" + datasetID + "/editions/" + edition + "/versions/" + version).
		Reply(http.StatusOK)

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
	return c.expectMinioFile(fileName, true, c.cfg.PublicBucketName)
}

// theFollowingPrivateFileCanBeSeenInMinio checks that the provided fileName is available in the private bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingPrivateFileCanBeSeenInMinio(fileName string) error {
	return c.expectMinioFile(fileName, true, c.cfg.PrivateBucketName)
}

// theFollowingPublicFileCannotBeSeenInMinio checks that the provided fileName is NOT available in the public bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingPublicFileCannotBeSeenInMinio(fileName string) error {
	return c.expectMinioFile(fileName, false, c.cfg.PublicBucketName)
}

// theFollowingPrivateFileCannotBeSeenInMinio checks that the provided fileName is NOT available in the private bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) theFollowingPrivateFileCannotBeSeenInMinio(fileName string) error {
	return c.expectMinioFile(fileName, false, c.cfg.PrivateBucketName)
}

// expectMinioFile checks that the provided fileName 'is' / 'is NOT' available in the provided bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) expectMinioFile(fileName string, expected bool, bucketName string) error {
	var b []byte
	f := aws.NewWriteAtBuffer(b)

	// probe bucket with backoff to give time for event to be processed
	retries := MinioCheckRetries
	timeout := time.Second
	var numBytes int64
	var err error

	for {
		numBytes, err = c.S3Downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
		})
		if err == nil || retries <= 0 {
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
		if !expected {
			// file was not expected - expected error is 'NoSuchKey'
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == s3.ErrCodeNoSuchKey {
					log.Info(c.ctx, "successfully checked that file not found in minio")
					return nil
				}
			}
			// file was not expected but error is different than 'NoSuchKey'
			return fmt.Errorf(
				"error checking that file was not present in minio. Last error: %w",
				err,
			)
		}
		// file was expected - return wrapped error
		return fmt.Errorf(
			"error obtaining file from minio. Last error: %w",
			err,
		)
	}

	// file was not expected but it was found
	if !expected {
		return errors.New("found unexpected file in minio")
	}

	// file was expected and found - validate size
	if numBytes < 1 {
		return errors.New("file length zero")
	}

	// log file content
	log.Info(c.ctx, "got file contents", log.Data{
		"contents (Note: this will look garbled for an xlsx file)": string(f.Bytes()),
	})

	return nil
}

func (c *Component) thisFileIsPutInPublicS3Bucket(fileName string, file *godog.DocString) error {
	return c.putFileInBucket(fileName, file, c.cfg.PublicBucketName)
}

func (c *Component) putFileInBucket(fileName string, file *godog.DocString, bucketName string) error {
	//!!! need to take into account private or public ?

	spew.Dump(file.Content)
	s := spew.Sdump(file.Content)

	log.Info(c.ctx, "file contents", log.Data{
		"contents": s,
	})
	fmt.Printf("\n\nfile: %s\n\n", s)

	fileReader, fileWriter := io.Pipe()

	wgUpload := sync.WaitGroup{}
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()

		// Write the 'in memory' stringt to the given io.writer
		if _, err := fileWriter.Write([]byte(file.Content)); err != nil {
			report := handler.NewError(err,
				log.Data{"err": err, "bucketName": bucketName, "filenameXlsx": fileName})

			if closeErr := fileWriter.CloseWithError(report); closeErr != nil {
				log.Error(c.ctx, "error closing upload writerWithError", closeErr)
			}
		} else {
			log.Info(c.ctx, fmt.Sprintf("finished writing file: %s, to pipe for bucket: %s", fileName, bucketName))

			if closeErr := fileWriter.Close(); closeErr != nil {
				log.Error(c.ctx, "error closing upload writer", closeErr)
			}
		}
	}()

	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &fileName,
		Body:   fileReader,
	}

	// Perform an upload.
	result, err := c.S3Uploader.Upload(upParams)

	_ = result

	/*	s = spew.Sdump(result)

		fmt.Printf("\n\nresult: %s\n\n", s)

		s = spew.Sdump(err)

		fmt.Printf("\n\nerr: %s\n\n", s)
	*/
	// The whole file has now been uploaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgUpload.Wait()

	return err
}
