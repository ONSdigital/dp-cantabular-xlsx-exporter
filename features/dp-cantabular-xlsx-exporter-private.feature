Feature: Cantabular-Xlsx-Exporter-Published

  # This file validates that an XLSX file generated for a ENCRYPTED csv file and a metadata structure is 
  # available via datasets api (a json object here) in published state are stored in the public S3 bucket

  Background:
	  Given the following Csv file named: "dataset-happy-01-edition-happy-01-version-happy-01.csv" is available as an ENCRYPTED file in Private S3 bucket for dataset-id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01":
      """
      count,City,Number of siblings (3 mappings),Sex
      1,London,No siblings,Male
      0,London,No siblings,Female
      0,London,1 or 2 siblings,Male
      0,London,1 or 2 siblings,Female
      0,London,3 or more siblings,Male
      1,London,3 or more siblings,Female
      0,Liverpool,No siblings,Male
      0,Liverpool,No siblings,Female
      0,Liverpool,1 or 2 siblings,Male
      0,Liverpool,1 or 2 siblings,Female
      1,Liverpool,3 or more siblings,Male
      0,Liverpool,3 or more siblings,Female
      0,Belfast,No siblings,Male
      0,Belfast,No siblings,Female
      1,Belfast,1 or 2 siblings,Male
      0,Belfast,1 or 2 siblings,Female
      0,Belfast,3 or more siblings,Male
      2,Belfast,3 or more siblings,Female
      """

    And the following metadata document with dataset id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01" is available from dp-dataset-api:
      """
      {
        "dimensions": [
          {
            "label": "City",
            "links": {
              "code_list": {},
              "options": {},
              "version": {}
            },
            "href": "http://api.localhost:23200/v1/code-lists/city",
            "id": "city",
            "name": "City"
          },
          {
            "label": "Number of siblings (3 mappings)",
            "links": {
              "code_list": {},
              "options": {},
              "version": {}
            },
            "href": "http://api.localhost:23200/v1/code-lists/siblings",
            "id": "siblings",
            "name": "Number of siblings (3 mappings)"
          },
          {
            "label": "Sex",
            "links": {
              "code_list": {},
              "options": {},
              "version": {}
            },
            "href": "http://api.localhost:23200/v1/code-lists/sex",
            "id": "sex",
            "name": "Sex"
          }
        ],
        "distribution": [
          "json",
          "csvw",
          "txt"
        ],
        "downloads": {},
        "release_date": "2021-11-19T00:00:00.000Z",
        "title": "Test Cantabular Dataset Published",
        "headers": [
          "cantabular_table",
          "city",
          "siblings_3",
          "sex"
        ]
      }
      """

    And the following Associated instance with id "instance-happy-01" is available from dp-dataset-api:
      """
      {
        "import_tasks": {
          "build_hierarchies": null,
          "build_search_indexes": null,
          "import_observations": {
            "total_inserted_observations": 0,
            "state": "created"
          }
        },
        "id": "057cd26b-e0ae-431f-9316-913db61cec39",
        "last_updated": "2021-07-19T09:59:28.417Z",
        "links": {
          "dataset": {
            "href": "http://localhost:22000/datasets/cantabular-dataset",
            "id": "cantabular-dataset"
          },
          "job": {
            "href": "http://localhost:21800/jobs/e7f99293-44f2-47ce-b6cb-db2f6618ef40",
            "id": "e7f99293-44f2-47ce-b6cb-db2f6618ef40"
          },
          "self": {
            "href": "http://10.201.4.160:10400/instances/057cd26b-e0ae-431f-9316-913db61cec39"
          }
        },
        "state": "associated",
        "headers": [
          "ftb_table",
          "city",
          "siblings"
        ],
        "is_based_on": {
          "@type": "cantabular_table",
          "@id": "Example"
        }
      }
      """

    # the PUT receiver step needs to be registered first in anticipation of the PUT happening because the service is running asynchronously to the test code
    And a PUT endpoint exists in dataset-API for dataset-id "dataset-happy-01", edition "edition-happy-01" and version "version-happy-01" to be later updated by an API call with:
      """
      {
        "alerts": null,
        "collection_id": "",
        "downloads": {
          "CSVW": {
            "href": "http://localhost:23600/downloads/datasets/cantabular-example-1/editions/2021/versions/1.csv-metadata.json",
            "size": "641",
            "public": "http://minio:9000/dp-cantabular-metadata-exporter-pub/datasets/cantabular-example-1-2021-1.csvw"
          },
          "TXT": {
            "href": "http://localhost:23600/downloads/datasets/cantabular-example-1/editions/2021/versions/1.txt",
            "size": "499",
            "public": "http://minio:9000/dp-cantabular-metadata-exporter-pub/datasets/cantabular-example-1-2021-1.txt"
          }
        },
        "edition": "",
        "dimensions": null,
        "id": "",
        "instance_id": "",
        "latest_changes": null,
        "links": {
          "access_rights": {
            "href": ""
          },
          "dataset": {
            "href": ""
          },
          "dimensions": {
            "href": ""
          },
          "edition": {
            "href": ""
          },
          "editions": {
            "href": ""
          },
          "latest_version": {
            "href": ""
          },
          "versions": {
            "href": ""
          },
          "self": {
            "href": ""
          },
          "code_list": {
            "href": ""
          },
          "options": {
            "href": ""
          },
          "version": {
            "href": ""
          },
          "code": {
            "href": ""
          },
          "taxonomy": {
            "href": ""
          },
          "job": {
            "href": ""
          }
        },
        "release_date": "",
        "state": "",
        "temporal": null,
        "version": 0
      }
      """
    
    And a GET endpoint exists in dataset-API for dataset-id "dataset-happy-01", edition "edition-happy-01":

    """
    {
    "items": [
        {
            "alerts": [],
            "collection_id": "",
            "dataset_id": "dataset-happy-01",
            "dimensions": [
                {
                    "description": "ladcd description",
                    "label": "Local Authority code",
                    "links": {
                        "code_list": {},
                        "options": {},
                        "version": {}
                    },
                    "href": "http://api.localhost:23200/v1/code-lists/ladcd",
                    "id": "ladcd",
                    "name": "ladcd",
                    "variable": "LADCD",
                    "number_of_options": 349,
                    "is_area_type": true
                },
                {
                    "label": "Occupancy rating for bedrooms (5 categories)",
                    "links": {
                        "code_list": {},
                        "options": {},
                        "version": {}
                    },
                    "href": "http://api.localhost:23200/v1/code-lists/occupancy_rating_bedrooms_5a",
                    "id": "occupancy_rating_bedrooms_5a",
                    "name": "occupancy_rating_bedrooms_5a",
                    "variable": "occupancy_rating_bedrooms_5a",
                    "number_of_options": 5,
                    "is_area_type": false
                }
            ],
            "downloads": {
                "csv": {
                    "href": "http://localhost:23600/downloads/datasets/dataset-happy-01/editions/edition-happy-01/versions/3.csv",
                    "size": "102171"
                },
                "csvw": {
                    "href": "http://localhost:23600/downloads/datasets/dataset-happy-01/editions/edition-happy-01/versions/3.csv-metadata.json",
                    "size": "962"
                },
                "txt": {
                    "href": "http://localhost:23600/downloads/datasets/dataset-happy-01/editions/edition-happy-01/versions/3.txt",
                    "public": "http://minio:9000/public-bucket/datasets/dataset-happy-01-3.txt",
                    "size": "8615"
                }
            },
            "edition": "2021",
            "id": "edition-happy-01",
            "latest_changes": [],
            "links": {
                "dataset": {
                    "href": "http://dp-dataset-api:22000/datasets/dataset-happy-01",
                    "id": "Andre-Local-Meta-Test-8"
                },
                "dimensions": {},
                "edition": {
                    "href": "http://localhost:22000/datasets/dataset-happy-01/editions/edition-happy-01",
                    "id": "2021"
                },
                "self": {
                    "href": "http://localhost:22000/datasets/dataset-happy-01/editions/edition-happy-01/versions/3"
                }
            },
            "release_date": "2022-10-26T00:00:00.000Z",
            "state": "published",
            "usage_notes": [],
            "is_based_on": {
                "@type": "cantabular_table",
                "@id": "Example"
            },
            "version": 3,
            "type": "cantabular_table"
        }
    ],
    "count": 1,
    "offset": 0,
    "limit": 100,
    "total_count": 1
    }
    """

    And I have these filter outputs:
    """
    [
      {
        "id": "filterOutput-happy-01",
        "filter_id": "74310d8d-72d6-492a-bc30-27584627edb3",
        "instance_id": "instance-happy-01",
        "state": "published",
        "dataset": {
          "edition":"2021",
          "id":"cantabular-flexible-example",
          "version": 1
        },
        "dimensions": [
          {
            "name": "silbings",
            "id": "siblings_3",
            "label": "Number Of Siblings (3 Mappings)",
            "options": [
              "0-3",
              "4-7",
              "7+"
            ],
            "is_area_type": false
          },
          {
            "name": "geography",
            "id": "city",
            "label": "City",
            "options": [
              "Cardiff",
              "London",
              "Swansea"
            ],
            "is_area_type": true
          }
        ],
        "etag": "testEtag",
        "events": null,
        "links":{
          "filter_blueprint":{
            "href":":27100/filters/74310d8d-72d6-492a-bc30-27584627edb3"
          },
          "self":{
            "href":":27100/filter-outputs/94310d8d-72d6-492a-bc30-27584627edb1"
          },
          "version":{
            "href":":27100/datasets/cantabular-flexible-example/editions/2021/versions/1"
          }
        },
        "population_type": "Example",
        "published": true,
        "type": "flexible",
        "downloads":
        {
          "csv":
          {
            "href":"http://localhost:23600/downloads/datasets/cantabular-flexible-example/editions/2021/versions/1.csv",
            "private":"http://minio:9000/private-bucket/datasets/cantabular-flexible-example-2021-1.csv",
            "public":"https://csv-exported.s3.eu-west-1.amazonaws.com/full-datasets/cpih01-time-series-v1.csv-metadata.csv",
            "size":"277",
            "skipped":true
          },
          "csvw":
          {
            "href" : "http://localhost:23600/downloads/datasets/cantabular-flexible-example/editions/2021/versions/1.csv-metadata.json",
            "private" : "http://minio:9000/private-bucket/datasets/cantabular-flexible-example-2021-1.csvw",
            "public" : "https://csv-exported.s3.eu-west-1.amazonaws.com/full-datasets/cpih01-time-series-v1.csv-metadata.csvw",
            "size" : "607",
            "skipped": true
          },
          "txt":
          {
            "href" : "http://localhost:23600/downloads/datasets/cantabular-flexible-example/editions/2021/versions/1.txt",
            "private" : "http://minio:9000/private-bucket/datasets/cantabular-flexible-example-2021-1.txt",
            "public" : "https://csv-exported.s3.eu-west-1.amazonaws.com/full-datasets/cpih01-time-series-v1.csv-metadata.txt",
            "size" : "530",
            "skipped": true
          }
        }
      }
    ]
    """

  Scenario: Consuming a cantabular-csv-created event with correct fields for a associated (unpublished/private) instance

    Given dp-dataset-api is healthy

    When the service starts

    Then this cantabular-csv-created event is queued, to be consumed:
      """
      {
        "InstanceID":     "instance-happy-01",
        "DatasetID":      "dataset-happy-01",
        "Edition":        "edition-happy-01",
        "Version":        "version-happy-01",
        "RowCount":       19,
        "FileName":       "dataset-happy-01-edition-happy-01-version-happy-01.csv",
        "FilterOutputID": "",
        "DimensionsID": []
      }
      """

    And a private file with filename "datasets/dataset-happy-01-edition-happy-01-version-happy-01.xlsx" can be seen in minio
    
    And the "xls" download in "filterOutputs" with key "id" value "filterOutput-happy-01" is updated with "datasets/dataset-happy-01-edition-happy-01-version-happy-01.xlsx"
    