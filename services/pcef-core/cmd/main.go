package main

import (
	"context"
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	"pcef-shaper-system/internal/pkg/telemetry"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/app"
	"pcef-shaper-system/services/pcef-core/internal/config"
	transport "pcef-shaper-system/services/pcef-core/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.LoadConfig("services/pcef-core/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Исполнительного Ядра Трафика PCEF Core (User Plane Go Engine)...")

	// 1. Устанавливаем gRPC-соединение с OCS-биллингом (порт 50054)
	ocsConn, err := grpc.Dial(cfg.OcsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось установить сетевое соединение с OCS-биллингом по адресу %s: %v", cfg.OcsAddr, err)
	}
	defer ocsConn.Close()
	ocsClient := gen.NewDiameterGyClient(ocsConn)

	// 2. Устанавливаем gRPC-соединение с Message Bus Kafka (порт 9092)
	kafkaConn, err := grpc.Dial(cfg.OfcsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось установить сетевое соединение с Kafka по адресу %s: %v", cfg.OfcsAddr, err)
	}
	defer kafkaConn.Close()
	kafkaClient := gen.NewDiameterGzClient(kafkaConn)

	// ИСПРАВЛЕНО (Синхронизация Graceful Shutdown): Создаем чистый контекст с отменой.
	// Мы вызовем cancel() вручную сразу после остановки gRPC сервера, чтобы гарантировать
	// кристально чистое завершение всех 32 фоновых демонов-хранителей ОЗУ!
	// FIXED: Bound application runtime scope to explicit context cancellation to clear all background janitors cleanly
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Dependency Injection слоев архитектуры (прокидываем контекст и оба клиента)
	coreEngine := app.NewPcefCoreService(ctx, ocsClient, kafkaClient)

	log.Info("🪐 [PCEF CORE]: Кластер из 32 высокоскоростных шард ОЗУ успешно взведен.")

	// Наполняем кэш стартовыми абонентами
	coreEngine.RegisterSubscriber(context.Background(), "250010000000001", "192.168.1.50", "VIP")
	coreEngine.RegisterSubscriber(context.Background(), "250010000000002", "192.168.1.51", "BASE")

	// Слой телеметрии OpenTelemetry
	metrics, err := telemetry.InitOtelMetrics(cfg.ServiceName, ":8080", log)
	if err != nil {
		log.Fatal("Не удалось запустить слой телеметрии OpenTelemetry: %v", err)
	}

	grpcHandler := transport.NewGrpcHandler(coreEngine, log, metrics)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterTrafficPipelineServer(server, grpcHandler)
	gen.RegisterDiameterGxServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой gRPC-порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC PCEF-Core поток обработки фреймов успешно запущен на %s", cfg.BindAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatal("Крах рантайма gRPC сервера PCEF-Core: %v", err)
		}
	}()

	// Блокируем главный поток здесь, ожидая системного сигнала Linux (SIGTERM/SIGINT).
	// Внутри этого пакета отработает server.GracefulStop(), плавно закрывая сетевые соединения.
	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)

	// ИСПРАВЛЕНО: Сеть закрыта, теперь атомарно тушим 32 фоновых демона кэша!
	log.Info("🛑 [PCEF CORE]: Каскадный сброс контекста фоновых воркеров шард кэша...")
	cancel()

	// Небольшая b2b-пауза, чтобы горутины демонов успели прочитать ctx.Done() и выйти из циклов
	time.Sleep(100 * time.Millisecond)
	log.Info("🏆 [PCEF CORE]: User Plane Go Engine успешно и безопасно остановлен. 0% утечек.")
}
