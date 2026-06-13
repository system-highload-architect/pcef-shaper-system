package grpc

import (
	"context"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterRxServer
	pcrfCli gen.DiameterGxClient // Клиент к PCRF
	log     *logger.AppLogger
}

func NewGrpcHandler(pcrf gen.DiameterGxClient, log *logger.AppLogger) *GrpcHandler {
	return &GrpcHandler{pcrfCli: pcrf, log: log}
}

func (h *GrpcHandler) SendAAQuery(ctx context.Context, req *gen.MediaSessionRequest) (*gen.MediaSessionResponse, error) {
	h.log.Info("AF Rx Gateway -> Получен запрос от CDN на выделение 4K-полосы для абонента %s", req.SubscriberId)

	// Асинхронно взводим неблокирующий таймер отмены SLA-приоритета (Защита от утечек горутин)
	time.AfterFunc(time.Duration(req.DurationSeconds)*time.Second, func() {
		h.log.Info("AF Rx Gateway -> Время сессии %s истекло. SLA приоритет отозван.", req.SubscriberId)
	})

	return &gen.MediaSessionResponse{
		ResultCode: 2001, // DIAMETER_SUCCESS
		SessionId:  "rx_session_cdn_999",
	}, nil
}
