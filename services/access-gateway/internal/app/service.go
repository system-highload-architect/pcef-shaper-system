package app

import (
	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
)

type GatewayService struct {
	pcefClient gen.TrafficPipelineClient
	log        *logger.AppLogger
}

func NewGatewayService(pcef gen.TrafficPipelineClient, log *logger.AppLogger) *GatewayService {
	return &GatewayService{pcefClient: pcef, log: log}
}

// HandleRadiusPacket эмулирует низкоуровневый разбор UDP RADIUS Accounting фрейма
func (g *GatewayService) HandleRadiusPacket(ip, imsi string) {
	g.log.Info("RADIUS UDP Listener -> Перехвачен пакет Accounting-Start для IP: %s [IMSI: %s]", ip, imsi)
	// На проде тут шла бы сессионная привязка ядра Linux
}
