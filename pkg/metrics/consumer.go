package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Consumer reads metric events from Kafka and writes them to Postgres.
type Consumer struct {
	consumer *kafka.Consumer
	db       *pgxpool.Pool
}

// NewConsumer creates a Consumer that reads from the given broker and writes to Postgres.
func NewConsumer(brokerAddr, groupID string, db *pgxpool.Pool) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": brokerAddr,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
		// Only commit offsets explicitly after a successful Postgres write.
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, fmt.Errorf("creating kafka consumer: %w", err)
	}

	if err := c.Subscribe(TopicMetricEvents, nil); err != nil {
		return nil, fmt.Errorf("subscribing to topic: %w", err)
	}

	return &Consumer{consumer: c, db: db}, nil
}

// Run starts the consume loop. It blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	log.Println("metric consumer started")
	for {
		select {
		case <-ctx.Done():
			return c.consumer.Close()
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Timeout is normal — it just means no messages are available right now.
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				log.Printf("consumer read error: %v", err)
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				log.Printf("failed to handle message: %v", err)
				continue
			}

			// Commit the offset only after the message has been successfully written
			// to Postgres. This ensures we never lose an event if the consumer crashes.
			if _, err := c.consumer.CommitMessage(msg); err != nil {
				log.Printf("failed to commit offset: %v", err)
			}
		}
	}
}

// handleMessage deserialises a Kafka message and writes it to Postgres.
func (c *Consumer) handleMessage(ctx context.Context, msg *kafka.Message) error {
	var event MetricEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshalling event: %w", err)
	}

	return c.writeEvent(ctx, event)
}

// writeEvent writes a MetricEvent to Postgres.
// The unique constraint on (experiment_id, user_id, event_type) means duplicate
// events are silently ignored, giving us idempotent writes.
func (c *Consumer) writeEvent(ctx context.Context, event MetricEvent) error {
	tx, err := c.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert the raw event. ON CONFLICT DO NOTHING deduplicates retried messages.
	_, err = tx.Exec(ctx, `
		INSERT INTO metric_events (experiment_id, user_id, variant_id, event_type, value, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (experiment_id, user_id, event_type) DO NOTHING
	`, event.ExperimentID, event.UserID, event.VariantID, event.EventType, event.Value, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("inserting metric event: %w", err)
	}

	// Upsert into the pre-aggregated table so the stats engine can query
	// counts and sums without scanning the full metric_events table.
	_, err = tx.Exec(ctx, `
		INSERT INTO agg_metrics (experiment_id, variant_id, event_type, n_events, sum_value, sum_sq_value, last_updated)
		VALUES ($1, $2, $3, 1, $4, $5, $6)
		ON CONFLICT (experiment_id, variant_id, event_type) DO UPDATE
		SET
			n_events      = agg_metrics.n_events + 1,
			sum_value     = agg_metrics.sum_value + EXCLUDED.sum_value,
			sum_sq_value  = agg_metrics.sum_sq_value + EXCLUDED.sum_sq_value,
			last_updated  = EXCLUDED.last_updated
	`, event.ExperimentID, event.VariantID, event.EventType,
		event.Value, event.Value*event.Value, time.Now())
	if err != nil {
		return fmt.Errorf("upserting agg_metrics: %w", err)
	}

	return tx.Commit(ctx)
}
