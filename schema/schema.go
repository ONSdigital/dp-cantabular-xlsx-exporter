package schema

import (
	"github.com/ONSdigital/dp-kafka/v4/avro"
)

var cantabularCsvCreated = `{
  "type": "record",
  "name": "common-output-created",
  "fields": [
    {"name": "instance_id",       "type": "string", "default": ""},
    {"name": "dataset_id",        "type": "string", "default": ""},
    {"name": "edition",           "type": "string", "default": ""},
    {"name": "version",           "type": "string", "default": ""},
    {"name": "row_count",         "type": "int",    "default": 0 },
    {"name": "file_name",         "type": "string", "default": ""},
    {"name": "filter_output_id",  "type": "string", "default": ""},
    {"name": "dimensions",        "type": {"type": "array", "items": "string"}, "default": [] }
  ]
}`

// CantabularCsvCreated the Avro schema for CSV exported messages.
var CantabularCsvCreated = &avro.Schema{
	Definition: cantabularCsvCreated,
}
