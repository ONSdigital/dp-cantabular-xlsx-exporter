Feature: Cantabular-Xlsx-Exporter-Unhealthy

  Background:
    Given dp-dataset-api is unhealthy
  
    Scenario: Not consuming cantabular-csv-created events, because a dependency is not healthy

    When the service starts
    
    And this cantabular-csv-created event is queued, to be consumed:
      """
      {
        "InstanceID": "instance-happy-02",
        "DatasetID":  "dataset-happy-02",
        "Edition":    "edition-happy-02",
        "Version":    "version-happy-02",
        "RowCount":   19
      }
      """

    # We check that both public and private files do not exist, because without dp-dataset-api working we
    # do not know what to expect
    Then no public file with filename "datasets/dataset-happy-02-edition-happy-02-version-happy-02.xlsx" can be seen in minio

    And no private file with filename "datasets/dataset-happy-02-edition-happy-02-version-happy-02.xlsx" can be seen in minio
