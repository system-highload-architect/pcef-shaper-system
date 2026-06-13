package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/interceptors"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/spr-storage/internal/app"
	"pcef-shaper-system/services/spr-storage/internal/config"
	transport "pcef-shaper-system/services/spr-storage/transport/grpc"

	"google.golang.org/grpc"
)

func main() {
	// 1. Инициализация конфигурационного шасси
	cfg := config.LoadConfig("config.yaml")

	// 2. Взвод кольцевого structured-логера
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Инициализация NoSQL СУБД ScyllaDB (SPR) Эмулятора...")

	// 3. Dependency Injection: сборка DDD-слоев
	storageCore := app.NewStorageService(32) // 32 шарда согласно ТЗ
	grpcHandler := transport.NewGrpcHandler(storageCore)

	// 4. Запуск gRPC-сервера со сквозным интерцептором Latency Tracking
	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryServerInterceptor(log)),
	)
	gen.RegisterSubscriptionRepositoryServer(server, grpcHandler)

	listener, err := net.Listen("tcp", cfg.BindAddr)
	if err != nil {
		log.Fatal("Не удалось открыть сетевой порт %s: %v", cfg.BindAddr, err)
	}

	go func() {
		log.Info("gRPC NoSQL-Сервер базы данных успешно запущен на %s", cfg.BindAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatal("Крах рантайма gRPC сервера: %v", err)
		}
	}()

	// 5. Включение Graceful Shutdown перехватчика сигналов ядра Linux
	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
