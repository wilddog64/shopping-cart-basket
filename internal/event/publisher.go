package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/user/shopping-cart-basket/internal/model"
	"go.uber.org/zap"
)

// Publisher publishes cart events to RabbitMQ
type Publisher struct {
	exchange string
	logger   *zap.Logger
	enabled  bool
	// In a real implementation, this would use rabbitmq-client-go
	// publisher *rabbitmq.Publisher
}

// NewPublisher creates a new event publisher
func NewPublisher(exchange string, logger *zap.Logger, enabled bool) *Publisher {
	return &Publisher{
		exchange: exchange,
		logger:   logger,
		enabled:  enabled,
	}
}

// Publish publishes an event to RabbitMQ
func (p *Publisher) Publish(ctx context.Context, event *model.EventEnvelope) error {
	if !p.enabled {
		p.logger.Debug("event publishing disabled, skipping",
			zap.String("eventType", event.Type),
			zap.String("eventId", event.ID),
		)
		return nil
	}

	// Serialize event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// In a real implementation, this would publish to RabbitMQ:
	// return p.publisher.Publish(ctx, &rabbitmq.PublishOptions{
	//     Exchange:   p.exchange,
	//     RoutingKey: event.Type,
	// }, data)

	// For now, just log the event
	p.logger.Info("publishing event",
		zap.String("eventType", event.Type),
		zap.String("eventId", event.ID),
		zap.String("correlationId", event.CorrelationID),
		zap.String("exchange", p.exchange),
		zap.String("routingKey", event.Type),
		zap.Int("size", len(data)),
	)

	return nil
}

// Close closes the publisher
func (p *Publisher) Close() error {
	// In a real implementation, close the RabbitMQ connection
	return nil
}

// NoopPublisher is a publisher that does nothing (for testing)
type NoopPublisher struct{}

// NewNoopPublisher creates a new no-op publisher
func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

// Publish does nothing
func (p *NoopPublisher) Publish(ctx context.Context, event *model.EventEnvelope) error {
	return nil
}
