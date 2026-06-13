package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/ocs-rating/internal/app"
	"pcef-shaper-system/services/ocs-rating/internal/config"
	transport "pcef-shaper-system/services/ocs-rating/transport/grpc"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig("services/ocs-rating/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Инициализация финтех-ядра онлайн-биллинга OCS (Aerospike HMA Emulator)...")

	ratingCore := app.NewOcsService()
	grpcHandler := transport.NewGrpcHandler(ratingCore)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterDiameterGyServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC OCS-Биллинг сервер успешно запущен на %s", cfg.BindAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatal("Крах рантайма gRPC сервера OCS: %v", err)
		}
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
