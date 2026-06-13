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

	// Подключаемся к базе профилей ScyllaDB (spr-storage)
	sprConn, _ := grpc.Dial(cfg.SprAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer sprConn.Close()
	sprClient := gen.NewSubscriptionRepositoryClient(sprConn)

	pcrfCore := app.NewPcrfService(sprClient)
	// Находим строчку сборки хэндлера и добавляем туда наш логер 'log'
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
