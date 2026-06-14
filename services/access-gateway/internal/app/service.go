package app

import (
	"context"

	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
)

type GatewayService struct {
	pcrfClient gen.DiameterGxClient // Клиент к PCRF движку по Gx
	log        *logger.AppLogger
}

func NewGatewayService(pcrf gen.DiameterGxClient, log *logger.AppLogger) *GatewayService {
	return &GatewayService{pcrfClient: pcrf, log: log}
}

// HandleRadiusPacket имитирует перехват UDP RADIUS фрейма и запускает gRPC-сессию Gx
func (g *GatewayService) HandleRadiusPacket(ctx context.Context, ip, imsi string) {
	g.log.Info("RADIUS UDP -> Перехвачен фрейм Accounting-Start [IMSI: %s, IP: %s]", imsi, ip)

	// Инициируем gRPC запрос доставки правил в мозг сети (PCRF)
	resp, err := g.pcrfClient.ProvisionPccRules(ctx, &gen.PccRulesProvision{
		Imsi:            imsi,
		IpAddress:       ip,
		ActiveRuleNames: []string{"DYNAMIC_BOOTSTRAP"},
	})

	if err != nil || resp.ResultCode != 2001 {
		g.log.Error("Gx Interface Failure -> Не удалось доставить RADIUS-сессию в PCRF: %v", err)
		return
	}

	g.log.Info("Gx Interface SUCCESS -> Сессия IMSI [%s] успешно авторизована в Control Plane сети", imsi)
}
