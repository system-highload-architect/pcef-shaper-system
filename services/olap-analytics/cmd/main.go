package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/olap-analytics/internal/app"
	"pcef-shaper-system/services/olap-analytics/internal/config"
	transport "pcef-shaper-system/services/olap-analytics/transport/grpc"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig("services/olap-analytics/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Аналитического Хранилища ClickHouse (OLAP Emulator)...")

	chCore := app.NewClickHouseEmulator(cfg.DataDiskPath, log)
	grpcHandler := transport.NewGrpcHandler(chCore)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterDiameterGzServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой gRPC-порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC ClickHouse-Сервер успешно запущен на %s", cfg.BindAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatal("Крах рантайма gRPC сервера: %v", err)
		}
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
