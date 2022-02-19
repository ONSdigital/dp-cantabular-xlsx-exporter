package steps

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/service"
	componenttest "github.com/ONSdigital/dp-component-test"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	dps3 "github.com/ONSdigital/dp-s3/v2"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/ONSdigital/log.go/v2/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/maxcnunes/httpfake"
)

const (
	ComponentTestGroup    = "component-test" // kafka group name for the component test consumer
	DrainTopicTimeout     = 5 * time.Second  // maximum time to wait for a topic to be drained
	DrainTopicMaxMessages = 1000             // maximum number of messages that will be drained from a topic
	MinioCheckRetries     = 3                // maximum number of retires to validate that a file is present in minio
)

var (
	BuildTime = "1625046891"
	GitCommit = "7434fe334d9f51b7239f978094ea29d10ac33b16"
	Version   = ""
)

type Component struct {
	componenttest.ErrorFeature
	apiFeature       *componenttest.APIFeature
	DatasetAPI       *httpfake.HTTPFake
	CantabularSrv    *httpfake.HTTPFake
	CantabularAPIExt *httpfake.HTTPFake
	s3Private        *dps3.Client
	s3Public         *dps3.Client
	producer         kafka.IProducer
	errorChan        chan error
	svc              *service.Service
	cfg              *config.Config
	wg               *sync.WaitGroup
	signals          chan os.Signal
	testETag         string
	ctx              context.Context
	generator        service.Generator
	vaultClient      *vault.Client
}

func NewComponent() *Component {
	return &Component{
		errorChan:        make(chan error),
		DatasetAPI:       httpfake.New(),
		CantabularSrv:    httpfake.New(),
		CantabularAPIExt: httpfake.New(),
		wg:               &sync.WaitGroup{},
		testETag:         "13c7791bafdbaaf5e6660754feb1a58cd6aaa892",
		ctx:              context.Background(),
	}
}

// initService initialises the server, the mocks and waits for the dependencies to be ready
func (c *Component) initService(ctx context.Context) error {
	// register interrupt signals
	c.signals = make(chan os.Signal, 1)
	signal.Notify(c.signals, os.Interrupt, syscall.SIGTERM)

	// Read config
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg.DatasetAPIURL = c.DatasetAPI.ResolveURL("")

	log.Info(ctx, "config used by component tests", log.Data{"cfg": cfg})

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Endpoint:         aws.String(cfg.LocalObjectStore),
		Region:           aws.String(cfg.AWSRegion),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	// S3 downloader to check minio files
	s, err := session.NewSession(s3Config)
	if err != nil {
		return fmt.Errorf("error creating aws session: %w", err)
	}

	c.s3Private = dps3.NewClientWithSession(cfg.PrivateBucketName, s)
	c.s3Public = dps3.NewClientWithSession(cfg.PublicBucketName, s)

	// producer for triggering test events that will be consumed by the service
	if c.producer, err = kafka.NewProducer(
		ctx,
		&kafka.ProducerConfig{
			BrokerAddrs:       cfg.KafkaConfig.Addr,
			Topic:             cfg.KafkaConfig.CsvCreatedTopic,
			MinBrokersHealthy: &cfg.KafkaConfig.ProducerMinBrokersHealthy,
			KafkaVersion:      &cfg.KafkaConfig.Version,
			MaxMessageBytes:   &cfg.KafkaConfig.MaxBytes,
		},
	); err != nil {
		return fmt.Errorf("error creating kafka producer: %w", err)
	}

	// start kafka logging go-routines
	c.producer.LogErrors(ctx)

	c.generator = &generator{}

	service.GetGenerator = func() service.Generator {
		return c.generator
	}

	c.vaultClient, err = vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, service.VaultRetries)
	if err != nil {
		return err
	}

	// Create service and initialise it
	c.svc = service.New()
	if err = c.svc.Init(c.ctx, cfg, BuildTime, GitCommit, Version); err != nil {
		return fmt.Errorf("unexpected service Init error in NewComponent: %w", err)
	}

	c.cfg = cfg

	// wait for producer to be initialised and consumer to be in consuming state
	<-c.producer.Channels().Initialised
	log.Info(ctx, "component-test kafka producer initialised")

	return nil
}

// startService starts the service under test and blocks until an error or an os interrupt is received.
// Then it closes the service (graceful shutdown)
func (c *Component) startService(ctx context.Context) error {
	if err := c.svc.Start(ctx, c.errorChan); err != nil {
		return fmt.Errorf("unexpected error while starting service: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		// blocks until an os interrupt or a fatal error occurs
		select {
		case err := <-c.errorChan:
			if errClose := c.svc.Close(ctx); errClose != nil {
				log.Warn(ctx, "error closing server during error handing", log.Data{"close_error": errClose})
			}
			log.Info(ctx, "this is a log timestamp for the following PANIC")

			panic(fmt.Errorf("unexpected error received from errorChan: %w", err))
		case sig := <-c.signals:
			log.Info(ctx, "os signal received", log.Data{"signal": sig})
		}
		if err := c.svc.Close(ctx); err != nil {
			log.Info(ctx, "this is a log timestamp for the following PANIC 2")
			panic(fmt.Errorf("unexpected error during service graceful shutdown: %w", err))
		}
	}()

	return nil
}

// drainTopic drains the provided topic and group of any residual messages between scenarios.
// Prevents future tests failing if previous tests fail unexpectedly and
// leave messages in the queue.
//
// A temporary batch consumer is used, that is created and closed within this func
// A maximum of DrainTopicMaxMessages messages will be drained from the provided topic and group.
//
// This method accepts a waitGroup pointer. If it is not nil, it will wait for the topic to be drained
// in a new go-routine, which will be added to the waitgroup. If it is nil, execution will be blocked
// until the topic is drained (or time out expires)
func (c *Component) drainTopic(ctx context.Context, topic, group string, wg *sync.WaitGroup) error {
	msgs := []kafka.Message{}

	kafkaOffset := kafka.OffsetOldest
	batchSize := DrainTopicMaxMessages
	batchWaitTime := DrainTopicTimeout
	drainer, err := kafka.NewConsumerGroup(
		ctx,
		&kafka.ConsumerGroupConfig{
			BrokerAddrs:   c.cfg.KafkaConfig.Addr,
			Topic:         topic,
			GroupName:     group,
			KafkaVersion:  &c.cfg.KafkaConfig.Version,
			Offset:        &kafkaOffset,
			BatchSize:     &batchSize,
			BatchWaitTime: &batchWaitTime,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating kafka consumer to drain topic: %w", err)
	}

	// register batch handler with 'drained channel'
	drained := make(chan struct{})
	if err := drainer.RegisterBatchHandler(
		ctx,
		func(ctx context.Context, batch []kafka.Message) error {
			defer close(drained)
			msgs = append(msgs, batch...)
			return nil
		},
	); err != nil {
		return fmt.Errorf("error creating kafka drainer: %w", err)
	}

	// start drainer consumer group
	if err := drainer.Start(); err != nil {
		log.Error(ctx, "error starting kafka drainer", err)
	}

	// start kafka logging go-routines
	drainer.LogErrors(ctx)

	// waitUntilDrained is a func that will wait until the batch is consumed or the timeout expires
	// (with 100 ms of extra time to allow any in-flight drain)
	waitUntilDrained := func() {
		drainer.StateWait(kafka.Consuming)
		log.Info(ctx, "drainer is consuming", log.Data{"topic": topic, "group": group})

		delay := time.NewTimer(DrainTopicTimeout + 100*time.Millisecond)
		select {
		case <-delay.C:
			log.Info(ctx, "drain timeout has expired (no messages drained)")
		case <-drained:
			log.Info(ctx, "message(s) have been drained")
			// Ensure timer is stopped and its resources are freed
			if !delay.Stop() {
				// if the timer has been stopped then read from the channel
				<-delay.C
			}
		}

		defer func() {
			log.Info(ctx, "drained topic", log.Data{
				"len":      len(msgs),
				"messages": msgs,
				"topic":    topic,
				"group":    group,
			})
		}()

		if err := drainer.Close(ctx); err != nil {
			log.Warn(ctx, "error closing drain consumer", log.Data{"err": err})
		}

		<-drainer.Channels().Closed
		log.Info(ctx, "drainer is closed")
	}

	// sync wait if wg is not provided
	if wg == nil {
		waitUntilDrained()
		return nil
	}

	// async wait if wg is provided
	wg.Add(1)
	go func() {
		defer wg.Done()
		waitUntilDrained()
	}()
	return nil
}

// Close kills the application under test, and then it shuts down the testing consumer and producer.
func (c *Component) Close() {
	// kill application
	c.signals <- os.Interrupt

	// wait for graceful shutdown to finish (or timeout)
	c.wg.Wait()

	// close producer
	if err := c.producer.Close(c.ctx); err != nil {
		log.Error(c.ctx, "error closing kafka producer", err)
	}

	// drain topics in parallel
	wg := &sync.WaitGroup{}
	if err := c.drainTopic(c.ctx, c.cfg.KafkaConfig.CsvCreatedTopic, c.cfg.KafkaConfig.CsvCreatedGroup, wg); err != nil {
		log.Error(c.ctx, "error draining topic", err, log.Data{
			"topic": c.cfg.KafkaConfig.CsvCreatedTopic,
			"group": c.cfg.KafkaConfig.CsvCreatedGroup,
		})
	}
	wg.Wait()
}

// Reset runs before each scenario. It re-initialises the service under test and the api mocks.
// Note that the service under test should not be started yet
// to prevent race conditions if it tries to call un-initialised dependencies (steps)
func (c *Component) Reset() error {
	if err := c.initService(c.ctx); err != nil {
		return fmt.Errorf("failed to initialise service: %w", err)
	}

	c.DatasetAPI.Reset()
	c.CantabularSrv.Reset()
	c.CantabularAPIExt.Reset()

	return nil
}
