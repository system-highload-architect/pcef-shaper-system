package main

import (
	"context"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/ue-emulator/internal/app"
	"pcef-shaper-system/services/ue-emulator/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ShutdownAdapter адаптирует генератор под интерфейс GracefulServer
type ShutdownAdapter struct {
	generator *app.TrafficGenerator
}

func (a *ShutdownAdapter) GracefulStop() {
	a.generator.StopLoadTest()
}

func main() {
	// 1. Загрузка конфигурационного шасси
	cfg := config.LoadConfig("config.yaml")

	// 2. Инициализация единого структурированного логера платформы
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Нагрузочного Генератора Трафика Абонентов (UE Emulator)...")

	// 3. Установка gRPC-соединения с исполнительным ядром PCEF-Core
	pcefConn, err := grpc.Dial(cfg.GatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось установить сетевое gRPC-соединение с PCEF-ядром по адресу %s: %v", cfg.GatewayAddr, err)
	}
	defer pcefConn.Close()
	pcefClient := gen.NewTrafficPipelineClient(pcefConn)

	// 4. ИСПРАВЛЕНО: Прокидываем логер шасси третьим аргументом в конструктор
	// FIXED: Injecting the shared chassis logger as the 3rd parameter to match SRS contracts
	trafficGen := app.NewTrafficGenerator(cfg.SimulatedSubscribers, pcefClient, log)

	// 5. Запуск неблокирующего Highload-штурма фреймами
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = trafficGen.StartLoadTest(ctx)

	// 6. Включение Graceful Shutdown перехватчика сигналов
	adapter := &ShutdownAdapter{generator: trafficGen}
	shutdown.ListenSignals(log, adapter, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
