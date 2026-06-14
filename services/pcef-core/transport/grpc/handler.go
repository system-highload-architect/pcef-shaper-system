package grpc

import (
	"context"
	"io"

	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/internal/pkg/ratelimit" // ИМПОРТИРУЕМ LOCK-FREE ЛИМИТЕР
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedTrafficPipelineServer
	// ДОБАВЛЯЕМ СТРОКУ (Встраиваем нереализованный сервер Gx для прохождения проверок компилятора):
	gen.UnimplementedDiameterGxServer

	service app.ShaperEngine
	log     *logger.AppLogger
	limiter *ratelimit.TokenBucketLimiter
}

func NewGrpcHandler(service app.ShaperEngine, log *logger.AppLogger) *GrpcHandler {
	return &GrpcHandler{
		service: service,
		log:     log,
		// Инициализируем лимитер: скорость 50 запросов в сек, максимальный всплеск — 100 токенов
		limiter: ratelimit.NewTokenBucketLimiter(50.0, 100.0),
	}
}

func (h *GrpcHandler) ProcessTrafficStream(stream gen.TrafficPipeline_ProcessTrafficStreamServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			frame, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return status.Errorf(codes.DataLoss, "Failed to read network frame socket: %v", err)
			}

			// ЭШЕЛОН ЗАЩИТЫ: Бьем по Lock-Free лимитеру частоты запросов!
			allowed, limitErr := h.limiter.Allow(ctx, frame.SourceIp)
			if !allowed || limitErr != nil {
				// Нарушитель обнаружен — мгновенно уничтожаем трафик на сетевой карте через XDP_DROP!
				_ = stream.Send(&gen.EnforcementVerdict{
					SourceIp: frame.SourceIp,
					Action:   "XDP_DROP", // Жесткая отсечка
				})
				continue
			}

			verdict, err := h.service.ProcessPacket(ctx, frame)
			if err != nil {
				h.log.Error("Ошибка конвейера диспетчеризации PCEF: %v", err)
				continue
			}

			if err := stream.Send(verdict); err != nil {
				return status.Errorf(codes.Unavailable, "Failed to send enforcement verdict: %v", err)
			}
		}
	}
}

// Добавляем к структуре GrpcHandler поддержку контракта DiameterGxServer
// (Убедись, что в cmd/main.go вызван gen.RegisterDiameterGxServer(server, grpcHandler))
func (h *GrpcHandler) ProvisionPccRules(ctx context.Context, req *gen.PccRulesProvision) (*gen.PccRulesAck, error) {
	tariff := "BASE"
	if len(req.ActiveRuleNames) > 0 && req.ActiveRuleNames[0] == "VIP_UNLIMITED" {
		tariff = "VIP"
	}

	// Динамически регистрируем сессию абонента в RAM ядра по RADIUS-триггеру!
	h.service.RegisterSubscriber(ctx, req.Imsi, req.IpAddress, tariff)

	h.log.Info("PCEF Core Gx -> Сессия Абонента [%s] успешно взведена в Reactive LRU Cache по сигналу PCRF", req.IpAddress)
	return &gen.PccRulesAck{IsEnforced: true, ResultCode: 2001}, nil
}
