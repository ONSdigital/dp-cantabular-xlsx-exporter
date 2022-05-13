package event

// CantabularCsvCreated provides an avro structure for an Input event
type CantabularCsvCreated struct {
	InstanceID string   `avro:"instance_id"`
	DatasetID  string   `avro:"dataset_id"`
	Edition    string   `avro:"edition"`
	Version    string   `avro:"version"`
	RowCount   int32    `avro:"row_count"`
	Dimensions []string `avro:"dimension_ids"`
}
