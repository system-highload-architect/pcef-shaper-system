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

type ShutdownAdapter struct {
	worker *app.CollectorWorker
}

func (a *ShutdownAdapter) GracefulStop() {
	a.worker.Stop()
}

func main() {
	cfg := config.LoadConfig("services/ofcs-collector/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск асинхронного OFCS сборщика CDR-логов...")

	// Подключаемся к Kafka (порт 9092)
	kafkaConn, _ := grpc.Dial("localhost:9092", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer kafkaConn.Close()
	kafkaClient := gen.NewDiameterGzClient(kafkaConn)

	// Подключаемся к ClickHouse (порт 8123)
	chConn, _ := grpc.Dial(cfg.ClickhouseAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer chConn.Close()
	chClient := gen.NewDiameterGzClient(chConn)

	worker := app.NewCollectorWorker(kafkaClient, chClient, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.StartPipeline(ctx)

	adapter := &ShutdownAdapter{worker: worker}
	shutdown.ListenSignals(log, adapter, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
