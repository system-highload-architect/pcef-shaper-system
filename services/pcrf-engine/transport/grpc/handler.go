package grpc

import (
	"context"

	"pcef-shaper-system/internal/pkg/logger" // Подключаем наше общее платформенное шасси
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcrf-engine/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterGxServer
	service app.PolicyEngine
	log     *logger.AppLogger // Вшиваем логер для аудита скомпилированных политик
}

func NewGrpcHandler(service app.PolicyEngine, log *logger.AppLogger) *GrpcHandler {
	return &GrpcHandler{
		service: service,
		log:     log,
	}
}

// ProvisionPccRules принимает запросы от PCEF-ядра и возвращает скомпилированные PCC-правила
func (h *GrpcHandler) ProvisionPccRules(ctx context.Context, req *gen.PccRulesProvision) (*gen.PccRulesAck, error) {
	if req.Imsi == "" {
		return nil, status.Error(codes.InvalidArgument, "IMSI identifier cannot be empty")
	}

	// Вызываем эшелон бизнес-логики (Use Cases)
	profile, err := h.service.CompileRules(ctx, req.Imsi)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "PCRF compilation engine failure: %v", err)
	}

	// Пишем скомпилированные b2b-правила в кольцевой лог ротации!
	h.log.Info("PCRF SUCCESS -> Для IMSI [%s] успешно скомпилирован тариф [%s]. Взведено правил: %d %v",
		profile.IMSI, profile.TariffClass, len(profile.RuleNames), profile.RuleNames)

	return &gen.PccRulesAck{
		IsEnforced: true,
		ResultCode: 2001, // DIAMETER_SUCCESS
	}, nil
}
