package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcrf-engine/internal/app"
	"pcef-shaper-system/services/pcrf-engine/internal/config"
	transport "pcef-shaper-system/services/pcrf-engine/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.LoadConfig("services/pcrf-engine/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Движка Сетевых Политик PCRF Engine (Control Plane)...")

	// 1. Подключаемся к базе профилей ScyllaDB (spr-storage)
	sprConn, err := grpc.Dial(cfg.SprAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось подключиться к SPR по адресу %s: %v", cfg.SprAddr, err)
	}
	defer sprConn.Close()

	sprClient := gen.NewSubscriptionRepositoryClient(sprConn)

	// TODO
	// 2. Подключаемся к исполнительному ядру pcef-core по Gx интерфейсу (порт 50052)
	pcefConn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось подключиться к PCEF-Core: %v", err)
	}
	defer pcefConn.Close()

	// ИСПРАВЛЕНО: Для отправки Gx правил создаем строго NewDiameterGxClient!
	pcefGxClient := gen.NewDiameterGxClient(pcefConn)

	// Собираем слои через DI
	pcrfCore := app.NewPcrfService(sprClient, pcefGxClient)
	grpcHandler := transport.NewGrpcHandler(pcrfCore, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterDiameterGxServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой gRPC-порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC PCRF-Engine сервер успешно запущен на %s", cfg.BindAddr)
		_ = server.Serve(listener)
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
