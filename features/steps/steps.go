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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/cucumber/godog"
)

// RegisterSteps maps the human-readable regular expressions to their corresponding funcs
func (c *Component) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the following response is available from Cantabular from the codebook "([^"]*)" using the GraphQL endpoint:$`, c.theFollowingQueryResponseIsAvailable)
	ctx.Step(`^the following instance with id "([^"]*)" is available from dp-dataset-api:$`, c.theFollowingInstanceIsAvailable)
	ctx.Step(`^an instance with id "([^"]*)" is updated to dp-dataset-api`, c.theFollowingInstanceIsUpdated)
	ctx.Step(`^the service starts`, c.theServiceStarts)
	ctx.Step(`^dp-dataset-api is healthy`, c.datasetAPIIsHealthy)
	ctx.Step(`^dp-dataset-api is unhealthy`, c.datasetAPIIsUnhealthy)
	ctx.Step(`^this cantabular-csv-created event is queued, to be consumed:$`, c.thisCantabularCsvCreatedEventIsQueued)
	//	ctx.Step(`^these common-output-created events are produced:$`, c.theseInstanceCompleteEventsAreProduced)
	ctx.Step(`^a file with filename "([^"]*)" can be seen in minio`, c.theFollowingFileCanBeSeenInMinio)
	ctx.Step(`^no file with filename "([^"]*)" can be seen in minio`, c.theFollowingFileCannotBeSeenInMinio)
}

// theServiceStarts starts the service under test in a new go-routine
// note that this step should be called only after all dependencies have been setup,
// to prevent any race condition, specially during the first healthcheck iteration.
func (c *Component) theServiceStarts() error {
	c.wg.Add(1)
	go c.startService(c.ctx)
	return nil
}

// datasetAPIIsHealthy generates a mocked healthy response for dataset API healthecheck
func (c *Component) datasetAPIIsHealthy() error {
	const res = `{"status": "OK"}`
	c.DatasetAPI.NewHandler().
		Get("/health").
		Reply(http.StatusOK).
		BodyString(res)
	return nil
}

// datasetAPIIsUnhealthy generates a mocked unhealthy response for dataset API healthecheck
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

// theFollowingQueryResposneIsAvailable generates a mocked response for Cantabular Server
// POST /graphql?query with the provided query
func (c *Component) theFollowingQueryResponseIsAvailable(name string, cb *godog.DocString) error {
	const urlQuery = `{
		dataset(name: "Example") {
		 table(variables: ["city", "siblings"]) {
		  dimensions {
		   count
		   variable {
			name
			label
		   }
		   categories {
			code
			label
		   }
		  }
		  values
		  error
		 }
		}
	   }`

	c.CantabularAPIExt.NewHandler().
		Post("/graphql?query=" + urlQuery).
		Reply(http.StatusOK).
		BodyString(cb.Content)

	return nil
}

// theseCommonOutputEventsAreProduced consumes kafka messages that are expected to be produced by the service under test
// and validates that they match the expected values in the test
/*func (c *Component) theseInstanceCompleteEventsAreProduced(events *godog.Table) error {
	expected, err := assistdog.NewDefault().CreateSlice(new(event.InstanceComplete), events)
	if err != nil {
		return fmt.Errorf("failed to create slice from godog table: %w", err)
	}

	var got []*event.InstanceComplete
	listen := true

	for listen {
		select {
		case <-time.After(c.waitEventTimeout):
			listen = false
		case <-c.consumer.Channels().Closer:
			return errors.New("closer channel closed")
		case msg, ok := <-c.consumer.Channels().Upstream:
			if !ok {
				return errors.New("upstream channel closed")
			}

			var e event.InstanceComplete
			var s = schema.InstanceComplete

			if err := s.Unmarshal(msg.GetData(), &e); err != nil {
				msg.Commit()
				msg.Release()
				return fmt.Errorf("error unmarshalling message: %w", err)
			}

			msg.Commit()
			msg.Release()

			got = append(got, &e)
		}
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		return fmt.Errorf("-got +expected)\n%s\n", diff)
	}

	return nil
}*/

func (c *Component) thisCantabularCsvCreatedEventIsQueued(input *godog.DocString) error {
	// testing kafka message that will be produced
	var testEvent event.CantabularCsvCreated
	if err := json.Unmarshal([]byte(input.Content), &testEvent); err != nil {
		return fmt.Errorf("error unmarshaling input to event: %w body: %s", err, input.Content)
	}

	log.Info(c.ctx, "event to marshal: ", log.Data{
		"event": testEvent,
	})

	// marshal and send message
	b, err := schema.CantabularCsvCreated.Marshal(testEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event from schema: %w", err)
	}

	log.Info(c.ctx, "marshalled event: ", log.Data{
		"event": b,
	})

	c.producer.Channels().Output <- b

	return nil
}

func (c *Component) theFollowingFileCanBeSeenInMinio(fileName string) error {
	return c.expectMinioFile(fileName, true)
}

func (c *Component) theFollowingFileCannotBeSeenInMinio(fileName string) error {
	return c.expectMinioFile(fileName, false)
}

func (c *Component) expectMinioFile(fileName string, expected bool) error {
	var b []byte
	f := aws.NewWriteAtBuffer(b)

	// probe bucket with backoff to give time for event to be processed
	retries := MinioCheckRetries
	timeout := time.Second
	var numBytes int64
	var err error

	for {
		numBytes, err = c.S3Downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(c.cfg.PublicBucketName),
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
		"contents": string(f.Bytes()),
	})

	return nil
}
