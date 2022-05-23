package event

// CantabularCsvCreated provides an avro structure for an Input event
type CantabularCsvCreated struct {
	InstanceID string   `avro:"instance_id"`
	DatasetID  string   `avro:"dataset_id"`
	Edition    string   `avro:"edition"`
	Version    string   `avro:"version"`
	RowCount   int32    `avro:"row_count"`
	FileName   string   `avro:"file_name"`
	Dimensions []string `avro:"dimensions"`
}
