module github.com/ONSdigital/dp-cantabular-xlsx-exporter

go 1.16

replace github.com/coreos/etcd => github.com/coreos/etcd v3.3.24+incompatible

require (
	github.com/ONSdigital/dp-api-clients-go v1.41.1
	github.com/ONSdigital/dp-component-test v0.6.0
	github.com/ONSdigital/dp-healthcheck v1.1.3
	github.com/ONSdigital/dp-kafka/v2 v2.4.2
	github.com/ONSdigital/dp-net v1.2.0
	github.com/ONSdigital/dp-s3 v1.7.0
	github.com/ONSdigital/dp-vault v1.2.0
	github.com/ONSdigital/log.go/v2 v2.0.9
	github.com/aws/aws-sdk-go v1.41.4
	github.com/aws/aws-sdk-go-v2 v1.9.2
	github.com/aws/aws-sdk-go-v2/config v1.8.3
	github.com/aws/aws-sdk-go-v2/credentials v1.4.3
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.5.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.16.1
	github.com/cucumber/godog v0.12.1
	github.com/google/go-cmp v0.5.6
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/maxcnunes/httpfake v1.2.1
	github.com/rdumont/assistdog v0.0.0-20201106100018-168b06230d14
	github.com/smartystreets/goconvey v1.6.6
	github.com/xuri/excelize/v2 v2.4.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
)
