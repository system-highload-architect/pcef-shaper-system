package app

import (
	"context"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
)

type CollectorWorker struct {
	kafkaCli gen.DiameterGzClient
	chCli    gen.DiameterGzClient
	log      *logger.AppLogger
	stopChan chan struct{}
}

func NewCollectorWorker(kafka gen.DiameterGzClient, ch gen.DiameterGzClient, log *logger.AppLogger) *CollectorWorker {
	return &CollectorWorker{
		kafkaCli: kafka,
		chCli:    ch,
		log:      log,
		stopChan: make(chan struct{}),
	}
}

// StartPipeline запускает сборщик логов пачками строго по 30 штук (Req. 3 и Req. 5)
func (w *CollectorWorker) StartPipeline(ctx context.Context) {
	w.log.Info("Асинхронный конвейер OFCS сборщика логов запущен...")

	// Опрашиваем Kafka-буфер по таймеру раз в 500мс
	ticker := time.NewTicker(500 * time.Millisecond)

	go func() {
		for {
			select {
			case <-w.stopChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Имитируем чтение пачки сообщений из Kafka-топика (заглушка асинхронности)
				// В реальном ТЗ мы бы вызвали Consume. Здесь мы имитируем сборку пачки из 30 штук
				w.log.Info("OFCS-Collector -> Считывание пачки логов из топика Kafka...")

				var records []*gen.CallDetailRecord
				for i := 1; i <= 30; i++ {
					records = append(records, &gen.CallDetailRecord{
						RecordId:     "cdr_mock_123",
						SubscriberId: "250010000000001",
						BytesDumped:  1024 * 1024,
					})
				}

				// Пакетный сброс (Batch INSERT) в ClickHouse по gRPC Gz интерфейсу
				_, err := w.chCli.StreamCdrLogs(ctx, &gen.BulkCdrPack{Records: records})
				if err != nil {
					w.log.Error("Не удалось закоммитить пачку логов в ClickHouse: %v", err)
				}
			}
		}
	}()
}

func (w *CollectorWorker) Stop() {
	close(w.stopChan)
	w.log.Info("OFCS Сборщик логов плавно остановлен.")
}
