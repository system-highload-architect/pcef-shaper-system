package grpc

import (
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
	service app.ShaperEngine
	log     *logger.AppLogger
	limiter *ratelimit.TokenBucketLimiter // Вшиваем L7-щит
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
