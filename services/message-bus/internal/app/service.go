package app

import (
	"context"
	"fmt"
	"sync"

	"pcef-shaper-system/services/message-bus/internal/domain"
)

type KafkaEmulator struct {
	mu     sync.Mutex
	topic  chan *domain.CdrEvent
	maxCap int
}

func NewKafkaEmulator(capacity int) *KafkaEmulator {
	return &KafkaEmulator{
		topic:  make(chan *domain.CdrEvent, capacity),
		maxCap: capacity,
	}
}

// Publish осуществляет мгновенную неблокирующую запись лога в очередь (Non-blocking L4 I/O)
func (k *KafkaEmulator) Publish(ctx context.Context, event *domain.CdrEvent) error {
	select {
	case k.topic <- event:
		return nil
	default:
		// Если буфер Kafka переполнен, отбрасываем лог, чтобы не блокировать ядро трафика
		// If the Kafka buffer overflows, drop the trace log to prevent core engine thread starvation
		return fmt.Errorf("message-bus anomaly: ring buffer partition overflow, message dropped")
	}
}

// ConsumeBatch выгребает логи пачками (Batching) для пакетного сброса в ClickHouse
func (k *KafkaEmulator) ConsumeBatch(ctx context.Context, batchSize int) ([]*domain.CdrEvent, error) {
	var batch []*domain.CdrEvent

	// Вычитываем логи из канала, пока не соберем полную пачку или пока канал не опустеет
	for i := 0; i < batchSize; i++ {
		select {
		case event := <-k.topic:
			batch = append(batch, event)
		default:
			// Если в очереди больше нет сообщений, возвращаем то, что успели собрать
			return batch, nil
		}
	}

	return batch, nil
}
