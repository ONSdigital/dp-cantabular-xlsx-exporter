package event

//!!! adjust the following to indicate that an xlsx has been produced OK
// InstanceComplete provides an avro structure for a Instance Complete event
type InstanceComplete struct {
	InstanceID     string `avro:"instance_id"`
	CantabularBlob string `avro:"cantabular_blob"`
}

//!!! the following should be renamed to CantabularCsvCreated
//!!! and fix the comment below to indicat that this is an input to this service
// CommonOutputCreated provides an avro structure for an Output Created event
type CommonOutputCreated struct {
	FilterID   string `avro:"filter_output_id"`
	FileURL    string `avro:"file_url"`
	InstanceID string `avro:"instance_id"`
	DatasetID  string `avro:"dataset_id"`
	Edition    string `avro:"edition"`
	Version    string `avro:"version"`
	Filename   string `avro:"filename"`
	RowCount   int32  `avro:"row_count"`
}
