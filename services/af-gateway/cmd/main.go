package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/af-gateway/internal/config"
	transport "pcef-shaper-system/services/af-gateway/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.LoadConfig("services/af-gateway/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Шлюза Контент-Провайдеров AF Gateway (Diameter Rx)...")

	pcrfConn, _ := grpc.Dial(cfg.PcrfAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer pcrfConn.Close()
	pcrfClient := gen.NewDiameterGxClient(pcrfConn)

	grpcHandler := transport.NewGrpcHandler(pcrfClient, log)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterDiameterRxServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой gRPC-порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC AF-Gateway успешно запущен на %s", cfg.BindAddr)
		_ = server.Serve(listener)
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
