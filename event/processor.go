package event

import (
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/config"
)

// Processor handles consuming and processing Kafka messages
type Processor struct {
	cfg config.Config
}

// NewProcessor returns a new Processor
func NewProcessor(cfg config.Config) *Processor {
	return &Processor{
		cfg: cfg,
	}
}
