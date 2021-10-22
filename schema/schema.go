package schema

import (
	"github.com/ONSdigital/dp-kafka/v2/avro"
)

//!!! TODO, adjust these as needed fo xlsx exporter
var instanceComplete = `{
  "type": "record",
  "name": "cantabular-dataset-instance-complete",
  "fields": [
    {"name": "instance_id",     "type": "string", "default": ""},
    {"name": "cantabular_blob", "type": "string", "default": ""}
  ]
}`

// InstanceComplete is the Avro schema for Instance Complete messages.
var InstanceComplete = &avro.Schema{
	Definition: instanceComplete,
}

var cantabularCsvCreated = `{
  "type": "record",
  "name": "common-output-created",
  "fields": [
    {"name": "filter_output_id", "type": "string", "default": ""},
    {"name": "file_url", "type": "string", "default": ""},
    {"name": "instance_id", "type": "string", "default": ""},
    {"name": "dataset_id", "type": "string", "default": ""},
    {"name": "edition", "type": "string", "default": ""},
    {"name": "version", "type": "string", "default": ""},
    {"name": "filename", "type": "string", "default": ""},
    {"name": "row_count", "type": "int", "default": 0}
  ]
}`

// CantabularCsvCreated the Avro schema for CSV exported messages.
var CantabularCsvCreated = &avro.Schema{
	Definition: cantabularCsvCreated,
}
