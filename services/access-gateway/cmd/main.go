package main

import (
	"context"
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
	log.Info("Запуск Сетевого Шлюза Доступа Access Gateway...")

	// Шлюз подключается к PCRF-engine по Gx интерфейсу (порт 50053)
	// БЫЛО: pcrfConn, err := grpc.Dial(cfg.PcrfCoreAddr, ...)
	// СТАЛО (Используем каноничное поле из структуры конфига шлюза):
	pcrfConn, err := grpc.Dial(cfg.PcefCoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Не удалось подключиться к PCRF по адресу %s: %v", cfg.PcefCoreAddr, err)
	}
	defer pcrfConn.Close()
	pcrfClient := gen.NewDiameterGxClient(pcrfConn)

	gateway := app.NewGatewayService(pcrfClient, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Имитируем RADIUS UDP сессию на порту 1813 в фоне с задержкой, чтобы остальные узлы успели встать в рантайм
	go func() {
		time.Sleep(3 * time.Second)
		log.Info("RADIUS UDP Сигнальный порт успешно запущен на :1813")
		gateway.HandleRadiusPacket(ctx, "192.168.1.50", "250010000000001") // VIP абонент
		gateway.HandleRadiusPacket(ctx, "192.168.1.51", "250010000000002") // BASE абонент
	}()

	server := grpc.NewServer()
	listener, _ := net.Listen("tcp", cfg.BindAddr)
	go func() { _ = server.Serve(listener) }()

	shutdown.ListenSignals(log, server, time.Duration(cfg.ShutdownTimeout)*time.Second)
}
