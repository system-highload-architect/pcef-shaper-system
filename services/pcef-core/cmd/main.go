package main

import (
	"context"
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
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
	// На проде K8s CoreDNS свяжет доменное имя автоматически
	kafkaConn, err := grpc.Dial(cfg.OfcsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось установить сетевое соединение с Kafka по адресу %s: %v", cfg.OfcsAddr, err)
	}
	defer kafkaConn.Close()
	kafkaClient := gen.NewDiameterGzClient(kafkaConn)

	// Dependency Injection слоев архитектуры (прокидываем оба клиента)
	coreEngine := app.NewPcefCoreService(ocsClient, kafkaClient)

	// Наполняем кэш стартовыми абонентами для локального старта (на проде это сделает Шлюз по Gx)
	coreEngine.RegisterSubscriber(context.Background(), "250010000000001", "192.168.1.50", "VIP")
	coreEngine.RegisterSubscriber(context.Background(), "250010000000002", "192.168.1.51", "BASE")

	grpcHandler := transport.NewGrpcHandler(coreEngine, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterTrafficPipelineServer(server, grpcHandler)
	gen.RegisterDiameterGxServer(server, grpcHandler) // Сервер приема Control Plane правил

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

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
