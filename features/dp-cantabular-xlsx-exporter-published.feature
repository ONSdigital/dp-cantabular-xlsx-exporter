Feature: Cantabular-Xlsx-Exporter-Published

  # This file validates that an XLSX file generated for a csv file and a metadata structure is available via datasets api (a json object here) in published state are stored in the public S3 bucket

  Background:
#    Given the following Csv file named: '??? fill in' is available in Public S3 bucket:

#    And the following Metadata response is available from dataset api with dataset-id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01":

#    And dp-dataset-api is healthy

#    And the following instance with id "instance-happy-01" is available from dp-dataset-api:

#    Scenario: Consuming a cantabular-csv-created event with correct fields

#    When the service starts

#    And this cantabular-csv-created event is queued, to be consumed:

#    Then a public file with filename "datasets/dataset-happy-01-edition-happy-01-version-happy-01.csv" can be seen in minio

#    And a dataset version with dataset-id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01" is updated by an API call to dp-dataset-api
