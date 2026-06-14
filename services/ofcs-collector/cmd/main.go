package main

import (
	"context"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/ofcs-collector/internal/app"
	"pcef-shaper-system/services/ofcs-collector/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ShutdownAdapter адаптирует интерфейс воркера под контракт пакета shutdown
type ShutdownAdapter struct {
	worker app.CdrPipelineOrchestrator // ИСПРАВЛЕНО: Зависим строго от абстракции
}

func (a *ShutdownAdapter) GracefulStop() {
	a.worker.Stop()
}

func main() {
	cfg := config.LoadConfig("services/ofcs-collector/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск асинхронного OFCS сборщика CDR-логов...")

	// 1. Устанавливаем соединения с внешними брокерами и базами данных
	kafkaConn, _ := grpc.Dial("localhost:9092", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer kafkaConn.Close()
	kafkaClient := gen.NewDiameterGzClient(kafkaConn)

	chConn, _ := grpc.Dial(cfg.ClickhouseAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer chConn.Close()
	chClient := gen.NewDiameterGzClient(chConn)

	// 2. ИСПРАВЛЕНО: Явное сопоставление реализации с интерфейсом Use Case слоя (Strict DI)
	// FIXED: Direct interface assignment enforcing compilation checks for Clean Architecture contracts
	var worker app.CdrPipelineOrchestrator = app.NewCollectorWorker(kafkaClient, chClient, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем асинхронный сборщик пакетов ClickHouse логов по интерфейсу
	worker.StartPipeline(ctx)

	// 3. Прокидываем интерфейсный адаптер в диспетчер сигналов Linux
	adapter := &ShutdownAdapter{worker: worker}
	shutdown.ListenSignals(log, adapter, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
