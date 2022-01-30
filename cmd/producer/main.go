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

	// Get Config
	cfg, err := config.Get()
	if err != nil {
		log.Fatal(ctx, "error getting config", err)
		os.Exit(1)
	}

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
		log.Fatal(ctx, "fatal error trying to create kafka producer", err, log.Data{"topic": cfg.KafkaConfig.CsvCreatedTopic})
		os.Exit(1)
	}

	// kafka error logging go-routines
	kafkaProducer.LogErrors(ctx)

	// Wait for producer to be initialised plus 500ms (this might not be needed as its not in csv exporter)
	<-kafkaProducer.Channels().Initialised
	time.Sleep(500 * time.Millisecond)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		e := scanEvent(scanner)
		log.Info(ctx, "sending cantabular-csv-created event", log.Data{"event": e})

		bytes, err := schema.CantabularCsvCreated.Marshal(e)
		if err != nil {
			log.Fatal(ctx, "failed to marshal event: %w", err)
			os.Exit(1)
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
