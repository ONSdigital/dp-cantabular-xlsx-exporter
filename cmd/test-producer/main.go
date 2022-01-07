package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/log.go/v2/log"
)

const serviceName = "dp-cantabular-xlsx-exporter"

func main() {
	log.Namespace = serviceName
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(ctx, "fatal runtime error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Get Config
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %s", err)
	}

	log.Info(ctx, "Starting Kafka Producer (messages read from stdin)", log.Data{"config": cfg})

	// Create Kafka Producer
	pConfig := &kafka.ProducerConfig{
		BrokerAddrs:     cfg.KafkaConfig.Addr,
		Topic:           cfg.KafkaConfig.CsvCreatedTopic,
		KafkaVersion:    &cfg.KafkaConfig.Version,
		MaxMessageBytes: &cfg.KafkaConfig.MaxBytes,
	}
	if cfg.KafkaConfig.SecProtocol == config.KafkaTLSProtocolFlag {
		pConfig.SecurityConfig = kafka.GetSecurityConfig(
			cfg.KafkaConfig.SecCACerts,
			cfg.KafkaConfig.SecClientCert,
			cfg.KafkaConfig.SecClientKey,
			cfg.KafkaConfig.SecSkipVerify,
		)
	}
	kafkaProducer, err := kafka.NewProducer(ctx, pConfig)
	if err != nil {
		return fmt.Errorf("failed to create kafka producer: %w", err)
	}

	// kafka error logging go-routines
	kafkaProducer.LogErrors(ctx)

	// Wait for producer to be initialised plus 500ms
	<-kafkaProducer.Channels().Initialised
	time.Sleep(500 * time.Millisecond)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		e := scanEvent(scanner)
		log.Info(ctx, "sending cantabular-csv-created event", log.Data{"event": e})

		bytes, err := schema.CantabularCsvCreated.Marshal(e)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		// Send bytes to Output channel
		kafkaProducer.Channels().Output <- bytes
	}
}

// scanEvent creates a CantabularCsvCreated event according to the user input
func scanEvent(scanner *bufio.Scanner) *event.CantabularCsvCreated {
	fmt.Println("--- [Send Kafka CantabularCsvCreated] ---")

	e := &event.CantabularCsvCreated{}

	fmt.Println("Please type the instance_id")
	fmt.Printf("$ ")
	scanner.Scan()
	e.InstanceID = scanner.Text()

	fmt.Println("Please type the dataset_id")
	fmt.Printf("$ ")
	scanner.Scan()
	e.DatasetID = scanner.Text()

	fmt.Println("Please type the edition")
	fmt.Printf("$ ")
	scanner.Scan()
	e.Edition = scanner.Text()

	fmt.Println("Please type the version")
	fmt.Printf("$ ")
	scanner.Scan()
	e.Version = scanner.Text()

	for {
		fmt.Println("Please type the row_count")
		fmt.Printf("$ ")
		scanner.Scan()
		i, err := strconv.ParseInt(scanner.Text(), 10, 32)
		if err != nil {
			fmt.Println("Wrong value provided for row_count. Value must be int32")
			continue
		}
		e.RowCount = int32(i)
		break
	}

	return e
}
