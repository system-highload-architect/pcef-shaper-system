package app

import (
	"context"
	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
)

type GatewayService struct {
	pcrfClient gen.DiameterGxClient
	log        *logger.AppLogger
}

func NewGatewayService(pcrf gen.DiameterGxClient, log *logger.AppLogger) *GatewayService {
	return &GatewayService{pcrfClient: pcrf, log: log}
}

// HandleRadiusPacket теперь четко реализует интерфейс RadiusSessionSignaling
func (g *GatewayService) HandleRadiusPacket(ctx context.Context, ip, imsi string) {
	g.log.Info("RADIUS UDP -> Перехвачен сигнальный фрейм Accounting-Start [IMSI: %s, IP: %s]", imsi, ip)

	resp, err := g.pcrfClient.ProvisionPccRules(ctx, &gen.PccRulesProvision{
		Imsi:            imsi,
		IpAddress:       ip,
		ActiveRuleNames: []string{"DYNAMIC_BOOTSTRAP"},
	})

	if err != nil || resp.ResultCode != 2001 {
		g.log.Error("Gx Interface Failure -> Ошибка gRPC-сигнализации Gx с PCRF: %v", err)
		return
	}

	g.log.Info("Gx Interface SUCCESS -> Сессия IP [%s] успешно авторизована в контуре ядра", ip)
}
