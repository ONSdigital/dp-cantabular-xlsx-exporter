Feature: Cantabular-Xlsx-Exporter-Published

  # This file validates that an XLSX file generated for a csv file and a metadata structure is available via datasets api (a json object here) in published state are stored in the public S3 bucket

  Background:
    Given the following Csv file named: '??? fill in' is available in Public S3 bucket:
    # !!! might need to mention the instance ID AND/OR the dataset ID, or it might be in the kafka message ???
    And the following Metadata file named: '?? fill in' is available in Public S3 bucket ??? or available via some API call ??? check !!!:
    And dp-dataset-api is healthy
    And ?? anything else needs to be healthy ?

    And the following instance with id "instance-happy-01" is available from dp-dataset-api:

    Scenario: Consuming a cantabular-csv-created event with correct fields

    When the service starts

    And this cantabular-csv-created event is queued, to be consumed:

    Then a dataset version with dataset-id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01" is updated by an API call to dp-dataset-api

# ??? any other steps here ??? !!!

    And a public file with filename "datasets/dataset-happy-01-edition-happy-01-version-happy-01.csv" can be seen in minio
