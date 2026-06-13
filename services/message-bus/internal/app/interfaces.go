package app

import (
	"context"

	"pcef-shaper-system/services/message-bus/internal/domain"
)

// MessageBroker описывает контракт управления неблокирующей очередью сообщений
// MessageBroker describes the contract for governing a non-blocking message queue topology
type MessageBroker interface {
	Publish(ctx context.Context, event *domain.CdrEvent) error
	ConsumeBatch(ctx context.Context, batchSize int) ([]*domain.CdrEvent, error)
}
