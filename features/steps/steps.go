package steps

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ONSdigital/dp-cantabular-filter-flex-api/model"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cucumber/godog"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	ctx.Step(`^the following Csv file named: "([^"]*)" is available as an ENCRYPTED file in Private S3 bucket for dataset-id "([^"]*)", edition "([^"]*)" and version "([^"]*)":$`, c.thisFileIsPutInPrivateS3Bucket)

	ctx.Step(`^the following Published instance with id "([^"]*)" is available from dp-dataset-api:$`, c.theFollowingInstanceIsAvailable)
	ctx.Step(`^the following Associated instance with id "([^"]*)" is available from dp-dataset-api:$`, c.theFollowingInstanceIsAvailable)
	ctx.Step(`^the service starts`, c.theServiceStarts)
	ctx.Step(`^dp-dataset-api is healthy`, c.datasetAPIIsHealthy)
	ctx.Step(`^dp-dataset-api is unhealthy`, c.datasetAPIIsUnhealthy)
	ctx.Step(`^a PUT endpoint exists in dataset-API for dataset-id "([^"]*)", edition "([^"]*)" and version "([^"]*)" to be later updated by an API call with:$`, c.theFollowingVersionWillBeUpdated)
	ctx.Step(`^a GET endpoint exists in dataset-API for dataset-id "([^"]*)", edition "([^"]*)":$`, c.theFollowingVersionsWillBeReturned)
	ctx.Step(`^this cantabular-csv-created event is queued, to be consumed:$`, c.thisCantabularCsvCreatedEventIsQueued)
	ctx.Step(`^a public file with filename "([^"]*)" can be seen in minio`, c.theFollowingPublicFileCanBeSeenInMinio)
	ctx.Step(`^a private file with filename "([^"]*)" can be seen in minio`, c.theFollowingPrivateFileCanBeSeenInMinio)
	ctx.Step(`^no public file with filename "([^"]*)" can be seen in minio`, c.theFollowingPublicFileCannotBeSeenInMinio)
	ctx.Step(`^no private file with filename "([^"]*)" can be seen in minio`, c.theFollowingPrivateFileCannotBeSeenInMinio)
	ctx.Step(`^I have these filter outputs:$`, c.iHaveTheseFilterOutputs)
	ctx.Step(`^the "([^"]*)" download in "([^"]*)" with key "([^"]*)" value "([^"]*)" is updated with "([^"]*)"`, c.theFollowingPrivateFileInfoIsSeenInFilterOutput)
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

// theFollowingVersionWillBeUpdated generates a mockedsresponse for dataset API
// PUT /datasets/{dataset_id}/editions/{edition}/versions/{version} with the provided update in the request body
func (c *Component) theFollowingVersionWillBeUpdated(datasetID, edition, version string, v *godog.DocString) error {
	url := fmt.Sprintf(
		"/datasets/%s/editions/%s/versions/%s",
		datasetID,
		edition,
		version,
	)

	c.DatasetAPI.NewHandler().
		Put(url).
		AssertCustom(newPutVersionAssertor([]byte(v.Content))).
		Reply(http.StatusOK)

	return nil
}

// theFollowingVersionsWillBeUpdated generates a mockedsresponse for dataset API
// PUT /datasets/{dataset_id}/editions/{edition}/versions/{version} with the provided update in the request body
func (c *Component) theFollowingVersionsWillBeReturned(datasetID, edition string, v *godog.DocString) error {
	url := fmt.Sprintf(
		"/datasets/%s/editions/%s/versions",
		datasetID,
		edition,
	)

	c.DatasetAPI.NewHandler().
		Get(url).
		AssertCustom(newPutVersionAssertor([]byte(v.Content))).
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

// theFollowingPrivateFileInfoIsSeenInFilterOutput checks if there is a file of given type in the given mongo collection
// for filter output with a specific ID in the given key collection.
func (c *Component) theFollowingPrivateFileInfoIsSeenInFilterOutput(fileType, col, key, fId, fileName string) error {
	ctx := context.Background()
	var bdoc primitive.D
	if err := c.store.Conn().Collection(col).FindOne(ctx, bson.M{key: fId}, &bdoc); err != nil {
		return errors.Wrap(err, "failed to retrieve document")
	}

	b, err := bson.MarshalExtJSON(bdoc, true, true)
	if err != nil {
		return errors.Wrap(err, "failed to marshal bson document")
	}

	var anyJson map[string]interface{}
	json.Unmarshal(b, &anyJson)

	// find downloads
	var collection = anyJson["downloads"].(map[string]interface{})
	for k, v := range collection {
		if k == fileType {
			s := reflect.ValueOf(v)
			if s.Type().Kind() == reflect.Map { // could have done with recursive call but we are not going that deep
				xlsCollection := v.(map[string]interface{})
				v := xlsCollection["public"].(string)
				if !strings.Contains(v, fileName) {
					return errors.Wrap(err, fmt.Sprintf("%s file not found in downlods in filter output under: %s", fileName, fileType))
				}
			}
		}
	}

	return nil
}

// expectMinioFile checks that the provided fileName 'is' / 'is NOT' available in the provided bucket.
// If it is not available it keeps checking following an exponential backoff up to MinioCheckRetries times.
func (c *Component) expectMinioFile(filename string, expected bool, bucketName string) error {
	var b = make([]byte, 10000) // limit the number of bytes read in
	f := aws.NewWriteAtBuffer(b)

	// probe bucket with backoff to give time for event to be processed
	retries := MinioCheckRetries
	timeout := time.Second
	var err error
	var s3ReadCloser io.ReadCloser

	for {
		if bucketName == c.cfg.PublicBucketName {
			s3ReadCloser, _, err = c.s3Public.Get(filename)
		} else {
			// As we are only interested in knowing that the file exists, we don't need to do GetWithPSK
			s3ReadCloser, _, err = c.s3Private.Get(filename)
		}

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
			if strings.Contains(err.Error(), s3.ErrCodeNoSuchKey) {
				log.Info(c.ctx, "successfully checked that file not found in minio")
				return nil
			}
			// file was not expected but error is different from 'NoSuchKey'
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

	numBytes, err := s3ReadCloser.Read(b)
	if errClose := s3ReadCloser.Close(); errClose != nil {
		log.Error(c.ctx, "error closing io.Closer", errClose)
	}

	if err != nil {
		return fmt.Errorf("file read problem: %w", err)
	}

	// file was expected and found - validate size
	if numBytes < 1 {
		return errors.New("file length zero")
	}

	if !strings.Contains(filename, "xlsx") { // skip Excel files as they look garbled
		// log file content
		log.Info(c.ctx, "got file contents", log.Data{
			"contents": string(f.Bytes()),
		})
	}

	return nil
}

func (c *Component) thisFileIsPutInPublicS3Bucket(fileName string, file *godog.DocString) error {
	return c.putFileInBucket(fileName, "", "", "", file, true, c.cfg.PublicBucketName)
}

func (c *Component) thisFileIsPutInPrivateS3Bucket(fileName, datasetID, edition, version string, file *godog.DocString) error {
	return c.putFileInBucket(fileName, datasetID, edition, version, file, false, c.cfg.PrivateBucketName)
}

func (c *Component) putFileInBucket(filename, datasetID, edition, version string, file *godog.DocString, isPublished bool, bucketName string) error {

	// create mechanism to present 'S3Uploader.Upload' with a 'reader'
	fileReader, fileWriter := io.Pipe()

	wgUpload := sync.WaitGroup{}
	wgUpload.Add(1)
	go func() {
		defer wgUpload.Done()

		// Write the 'in memory' straight to the given io.writer
		if _, err := fileWriter.Write([]byte(file.Content)); err != nil {
			report := handler.NewError(err,
				log.Data{"err": err, "bucketName": bucketName, "filenameXlsx": filename})

			if closeErr := fileWriter.CloseWithError(report); closeErr != nil {
				log.Error(c.ctx, "error closing upload writerWithError", closeErr)
			}
		} else {
			log.Info(c.ctx, fmt.Sprintf("finished writing file: %s, to pipe for bucket: %s", filename, bucketName))

			if closeErr := fileWriter.Close(); closeErr != nil {
				log.Error(c.ctx, "error closing upload writer", closeErr)
			}
		}
	}()

	// Use the Upload function to read from the io.Pipe() Writer:
	err := c.uploadFile(fileReader, isPublished, bucketName, filename, datasetID, edition, version)
	if err != nil {
		if closeErr := fileWriter.Close(); closeErr != nil {
			log.Error(c.ctx, "error closing upload writer", closeErr)
		}

		err = handler.NewError(fmt.Errorf("failed to upload .xlsx file to S3 bucket: %w", err),
			log.Data{"bucket": bucketName})
	}

	// The whole file has now been uploaded OK
	// We wait until any logs coming from the go routine have completed before doing anything
	// else to ensure the logs appear in the log file in the correct order.
	wgUpload.Wait()

	return err
}

func (c *Component) uploadFile(fileReader io.Reader, isPublished bool, bucketName, filename, datasetID, edition, version string) error {
	f := fmt.Sprintf("datasets/%s", filename)
	// Upload input parameters
	upParams := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &f,
		Body:   fileReader,
	}

	var err error
	logData := log.Data{
		"bucket":       bucketName,
		"filename":     filename,
		"is_published": isPublished,
	}

	if isPublished {

		// Perform an upload.
		_, err = c.s3Public.UploadWithContext(c.ctx, upParams)
		if err != nil {
			return handler.NewError(fmt.Errorf("UploadWithContext failed to write file: %w", err),
				logData,
			)
		}

	} else {
		psk, err := c.generator.NewPSK()
		if err != nil {
			return handler.NewError(fmt.Errorf("NewPSK failed to generate a PSK for encryption: %w", err),
				logData,
			)
		}

		vaultPath := fmt.Sprintf("%s/%s", c.cfg.VaultPath, filename)
		vaultKey := "key"

		log.Info(c.ctx, "writing key to vault", log.Data{"vault_path": vaultPath})

		if err := c.vaultClient.WriteKey(vaultPath, vaultKey, hex.EncodeToString(psk)); err != nil {
			return handler.NewError(fmt.Errorf("WriteKey failed to write key to vault: %w", err),
				logData,
			)
		}

		// This code needs to use 'UploadWithPSKAndContext', because when processing an Excel file that is
		// nearly 1 million lines it has been seen to take over 45 seconds and if nomad has instructed a service
		// to shut down gracefully before installing a new version of this app, then without using context this
		// could cause problems.
		_, err = c.s3Private.UploadWithPSKAndContext(c.ctx, upParams, psk)
		if err != nil {
			return handler.NewError(fmt.Errorf("UploadWithPSKAndContext failed to upload encrypted file to S3: %w", err),
				logData,
			)
		}
	}

	return err
}

func (c *Component) iHaveTheseFilterOutputs(docs *godog.DocString) error {
	ctx := context.Background()
	var filterOutputs []model.FilterOutput

	err := json.Unmarshal([]byte(docs.Content), &filterOutputs)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshall")
	}

	store := c.store
	col := c.cfg.FilterOutputsCollection

	for _, f := range filterOutputs {
		if _, err = store.Conn().Collection(col).UpsertById(ctx, f.ID, bson.M{"$set": f}); err != nil {
			return errors.Wrap(err, "failed to upsert filter output")
		}
	}

	return nil
}
