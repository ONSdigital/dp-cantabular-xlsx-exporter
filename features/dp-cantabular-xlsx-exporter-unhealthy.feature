Feature: Cantabular-Xlsx-Exporter-Unhealthy

  Background:
    Given dp-dataset-api is unhealthy
  
    Scenario: Not consuming cantabular-csv-created events, because a dependency is not healthy

    When the service starts
    
    And this cantabular-csv-created event is queued, to be consumed:
      """
      {
        "InstanceID": "instance-happy-01",
        "DatasetID":  "dataset-happy-01",
        "Edition":    "edition-happy-01",
        "Version":    "version-happy-01",
        "RowCount":   123
      }
      """

    Then no file with filename "datasets/dataset-happy-01-edition-happy-01-version-happy-01.xlsx" can be seen in minio
