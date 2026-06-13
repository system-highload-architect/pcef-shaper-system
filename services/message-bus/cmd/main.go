package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/message-bus/internal/app"
	"pcef-shaper-system/services/message-bus/internal/config"
	transport "pcef-shaper-system/services/message-bus/transport/grpc"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig("services/message-bus/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Инициализация шины данных оффлайн-логов Message Bus (Apache Kafka Go Emulator)...")

	kafkaCore := app.NewKafkaEmulator(cfg.QueueCapacity)
	grpcHandler := transport.NewGrpcHandler(kafkaCore)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterDiameterGzServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой gRPC-порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC Kafka-Шина данных успешно запущена на %s", cfg.BindAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatal("Крах рантайма gRPC сервера Kafka: %v", err)
		}
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
