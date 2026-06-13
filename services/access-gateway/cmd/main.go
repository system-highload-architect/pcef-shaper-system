package main

import (
	"net"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/shutdown"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/access-gateway/internal/app"
	"pcef-shaper-system/services/access-gateway/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.LoadConfig("services/access-gateway/config.yaml")
	log := logger.NewAppLogger(cfg.ServiceName, cfg.LogLevel)
	log.Info("Запуск Сетевого Шлюза Доступа Access Gateway (BNG/UPF Emulation)...")

	pcefConn, _ := grpc.Dial(cfg.PcefCoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer pcefConn.Close()
	pcefClient := gen.NewTrafficPipelineClient(pcefConn)

	gateway := app.NewGatewayService(pcefClient, log)

	// Эмулируем RADIUS UDP сессию на порту 1813 в фоне
	go func() {
		log.Info("RADIUS UDP Сигнальный порт успешно запущен на :1813")
		gateway.HandleRadiusPacket("192.168.1.50", "250010000000001")
	}()

	server := grpc.NewServer()
	// Шлюз работает как сквозная L4-труба, поднимаем пустой gRPC сервер для соответствия шасси K8s
	listener, _ := net.Listen("tcp", cfg.BindAddr)
	go func() {
		log.Info("gRPC Access-Gateway контроллер запущен на %s", cfg.BindAddr)
		_ = server.Serve(listener)
	}()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
