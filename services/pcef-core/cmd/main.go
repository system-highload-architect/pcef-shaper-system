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

	// Инициализируем высокопроизводительный gRPC-клиент к ocs-rating биллингу
	// На проде K8s CoreDNS прозрачно свяжет это доменное имя с нужным IP ноды
	ocsConn, err := grpc.Dial(cfg.OcsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось установить сетевое соединение с OCS-биллингом по адресу %s: %v", cfg.OcsAddr, err)
	}
	defer ocsConn.Close()
	ocsClient := gen.NewDiameterGyClient(ocsConn)

	// Dependency Injection слоев архитектуры
	coreEngine := app.NewPcefCoreService(ocsClient)

	// Наполняем кэш тестовыми абонентами (Имитируем RADIUS-авторизацию шлюза)
	coreEngine.RegisterSubscriber(context.Background(), "250010000000001", "192.168.1.50", "VIP")
	coreEngine.RegisterSubscriber(context.Background(), "250010000000002", "192.168.1.51", "BASE")

	grpcHandler := transport.NewGrpcHandler(coreEngine, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterTrafficPipelineServer(server, grpcHandler)

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
