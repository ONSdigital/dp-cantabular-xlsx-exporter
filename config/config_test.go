package config

import (
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("Given an environment with no environment variables set", t, func() {
		os.Clearenv()
		cfg, err := Get()

		Convey("When the config values are retrieved", func() {

			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, "localhost:26800")
				So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(cfg.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
				So(cfg.KafkaAddr, ShouldHaveLength, 1)
				So(cfg.KafkaAddr[0], ShouldEqual, "localhost:9092")
				So(cfg.KafkaVersion, ShouldEqual, "1.0.2")
				So(cfg.KafkaOffsetOldest, ShouldEqual, true)
				So(cfg.KafkaNumWorkers, ShouldEqual, 1)
				So(cfg.CsvCreatedGroup, ShouldEqual, "dp-cantabular-xlsx-exporter")
				So(cfg.CsvCreatedTopic, ShouldEqual, "cantabular-csv-created")
				So(cfg.CantabularOutputCreatedTopic, ShouldEqual, "cantabular-output-created")
				So(cfg.EncryptionDisabled, ShouldBeFalse)
				So(cfg.VaultPath, ShouldEqual, "secret/shared/psk")
				So(cfg.VaultAddress, ShouldEqual, "http://localhost:8200")
				So(cfg.VaultToken, ShouldEqual, "")
				So(cfg.DownloadServiceURL, ShouldEqual, "http://localhost:23600")
				So(cfg.AWSRegion, ShouldEqual, "eu-west-1")
				So(cfg.UploadBucketName, ShouldEqual, "dp-cantabular-csv-exporter")
				So(cfg.LocalObjectStore, ShouldEqual, "")
				So(cfg.MinioAccessKey, ShouldEqual, "")
				So(cfg.MinioSecretKey, ShouldEqual, "")
				So(cfg.OutputFilePath, ShouldEqual, "/tmp/helloworld.txt")
			})

			Convey("Then a second call to config should return the same config", func() {
				newCfg, newErr := Get()
				So(newErr, ShouldBeNil)
				So(newCfg, ShouldResemble, cfg)
			})
		})
	})
}
