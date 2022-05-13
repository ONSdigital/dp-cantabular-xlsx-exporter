package schema

import (
	"github.com/ONSdigital/dp-kafka/v3/avro"
)

var cantabularCsvCreated = `{
  "type": "record",
  "name": "common-output-created",
  "fields": [
    {"name": "instance_id", "type": "string", "default": ""},
    {"name": "dataset_id", "type": "string", "default": ""},
    {"name": "edition", "type": "string", "default": ""},
    {"name": "version", "type": "string", "default": ""},
    {"name": "row_count", "type": "int", "default": 0},
    {
      "name": "dimension_ids",
      "type": {
        "type": "array", 
        "items": {
          "type": "string"
        }
      }, "default": []
    }
  ]
}`

// CantabularCsvCreated the Avro schema for CSV exported messages.
var CantabularCsvCreated = &avro.Schema{
	Definition: cantabularCsvCreated,
}
