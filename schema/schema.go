package schema

import (
	"github.com/ONSdigital/dp-kafka/v2/avro"
)

var cantabularCsvCreated = `{
  "type": "record",
  "name": "common-output-created",
  "fields": [
    {"name": "instance_id", "type": "string", "default": ""},
    {"name": "dataset_id", "type": "string", "default": ""},
    {"name": "edition", "type": "string", "default": ""},
    {"name": "version", "type": "string", "default": ""},
    {"name": "row_count", "type": "int", "default": 0}
  ]
}`

// CantabularCsvCreated the Avro schema for CSV exported messages.
var CantabularCsvCreated = &avro.Schema{
	Definition: cantabularCsvCreated,
}
