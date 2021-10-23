package event

import (
	"context"
	"fmt"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/schema"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/log.go/v2/log"
)

// Consume converts messages to event instances, and pass the event to the provided handler.
func (p *Processor) Consume(ctx context.Context, cg kafka.IConsumerGroup, h Handler) {
	// consume loop, to be executed by each worker
	var consume = func(workerID int) {
		for {
			select {
			case msg, ok := <-cg.Channels().Upstream:
				if !ok {
					log.Info(ctx, "closing event consumer loop because upstream channel is closed", log.Data{"worker_id": workerID})
					return
				}

				msgCtx, cancel := context.WithCancel(ctx)

				if err := p.processMessage(msgCtx, msg, h); err != nil {
					log.Error(
						msgCtx,
						"failed to process message", //!!! should this mention that its also committing the message like what happens at the end of the function processMessage() ?
						err,
						log.Data{
							"log_data":    unwrapLogData(err),
							"status_code": statusCode(err),
						},
					)
				}
				msg.Release()
				cancel()
			case <-cg.Channels().Closer:
				log.Info(ctx, "closing event consumer loop because closer channel is closed", log.Data{"worker_id": workerID})
				return
			case <-ctx.Done():
				log.Info(ctx, "parent context closed - closing event consumer loop ", log.Data{"worker_id": workerID})
				return
			}
		}
	}

	// workers to consume messages in parallel
	for w := 1; w <= p.cfg.KafkaNumWorkers; w++ {
		go consume(w)
	}
}

// processMessage unmarshals the provided kafka message into an event and calls the handler.
// After the message is handled, it is committed.
func (p *Processor) processMessage(ctx context.Context, msg kafka.Message, h Handler) error {
	defer msg.Commit()

	var event CantabularCsvCreated
	s := schema.CantabularCsvCreated

	if err := s.Unmarshal(msg.GetData(), &event); err != nil {
		return &Error{
			err: fmt.Errorf("failed to unmarshal event: %w", err),
			logData: map[string]interface{}{
				"msg_data": msg.GetData(),
			},
		}
	}

	log.Info(ctx, "event received", log.Data{"event": event})

	// handle - commit on failure (implement error handling to not commit if message needs to be consumed again)
	if err := h.Handle(ctx, &event); err != nil {
		return fmt.Errorf("failed to handle event: %w", err)
	}

	log.Info(ctx, "event successfully processed - committing message", log.Data{"event": event})
	return nil
}
