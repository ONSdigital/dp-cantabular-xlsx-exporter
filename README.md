# dp-cantabular-xlsx-exporter
Cantabular xlsx exporter

### Getting started

* Run `make debug`

The service runs in the background consuming messages from Kafka.
An example event can be created using the helper script, `make produce`.

### Dependencies

* Requires running…
  * [kafka](https://github.com/ONSdigital/dp/blob/main/guides/INSTALLING.md#prerequisites)
* No further dependencies other than those defined in `go.mod`

### Configuration

| Environment variable                | Default                              | Description
| ----------------------------------- | ------------------------------------ | -----------
| BIND_ADDR                           | localhost:26800                      | The host and port to bind to
| GRACEFUL_SHUTDOWN_TIMEOUT           | 5s                                   | The graceful shutdown timeout in seconds (`time.Duration` format)
| HEALTHCHECK_INTERVAL                | 30s                                  | Time between self-healthchecks (`time.Duration` format)
| HEALTHCHECK_CRITICAL_TIMEOUT        | 90s                                  | Time to wait until an unhealthy dependent propagates its state to make this app unhealthy (`time.Duration` format)
| SERVICE_AUTH_TOKEN                  |                                      | The service token for this app
| DATASET_API_URL                     | http://localhost:22000               | The Dataset API URL
| DOWNLOAD_SERVICE_URL                | http://localhost:23600               | The Download Service URL, only used to generate download links
| AWS_REGION                          | eu-west-1                            | The AWS region to use
| UPLOAD_BUCKET_NAME                  | public-bucket                        | The name of the S3 bucket to store published xlsx files
| PRIVATE_UPLOAD_BUCKET_NAME          | private-bucket                       | The name of the S3 bucket to store un-published xlsx files
| LOCAL_OBJECT_STORE                  |                                      | Used during feature tests
| MINIO_ACCESS_KEY                    |                                      | Used during feature tests
| MINIO_SECRET_KEY                    |                                      | Used during feature tests
| VAULT_TOKEN                         |                                      | Use `make debug` to set a vault token
| VAULT_ADDR                          | http://localhost:8200                | The address of vault
| VAULT_PATH                          | secret/shared/psk                    | The vault path to store psks
| COMPONENT_TEST_USE_LOG_FILE         | false                                | Used during feature tests
| STOP_CONSUMING_ON_UNHEALTHY         | true                                 | Flag to enable/disable kafka-consumer consumption depending on health status. If true, the consumer will stop consuming on 'WARNING' and 'CRITICAL' and it will start consuming on 'OK'
| KAFKA_ADDR                          | localhost:9092                       | The kafka broker addresses (can be comma separated)
| KAFKA_VERSION                       | "1.0.2"                              | The kafka version that this service expects to connect to
| KAFKA_OFFSET_OLDEST                 | true                                 | Start processing Kafka messages in order from the oldest in the queue
| KAFKA_MAX_BYTES                     | 2000000                              | the maximum number of bytes per kafka message
| KAFKA_SEC_PROTO                     | _unset_                              | if set to `TLS`, kafka connections will use TLS [[1]](#notes_1)
| KAFKA_SEC_CA_CERTS                  | _unset_                              | CA cert chain for the server cert [[1]](#notes_1)
| KAFKA_SEC_CLIENT_KEY                | _unset_                              | PEM for the client key [[1]](#notes_1)
| KAFKA_SEC_CLIENT_CERT               | _unset_                              | PEM for the client certificate [[1]](#notes_1)
| KAFKA_SEC_SKIP_VERIFY               | false                                | ignores server certificate issues if `true` [[1]](#notes_1)
| CSV_CREATED_GROUP                   | dp-cantabular-xlsx-exporter          | The consumer group for this application to consume CantabularCsvCreated messages
| CSV_CREATED_TOPIC                   | cantabular-csv-created               | The name of the topic to consume messages from
| OTEL_EXPORTER_OTLP_ENDPOINT         | localhost:4317                       | Endpoint for OpenTelemetry service
| OTEL_SERVICE_NAME                   | dp-cantabular-xlsx-exporter          | Label of service for OpenTelemetry service
| OTEL_BATCH_TIMEOUT                  | 5s                                   | Timeout for OpenTelemetry

**Notes:**

    1. <a name="notes_1">For more info, see the [kafka TLS examples documentation](https://github.com/ONSdigital/dp-kafka/tree/main/examples#tls)</a>

### Healthcheck

 The `/health` endpoint returns the current status of the service. Dependent services are health checked on an interval defined by the `HEALTHCHECK_INTERVAL` environment variable.

 On a development machine a request to the health check endpoint can be made by:

 `curl localhost:8125/health`

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2021, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details

