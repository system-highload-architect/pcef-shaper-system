package grpc

import (
	"context"
	"time"

	"pcef-shaper-system/internal/pkg/logger"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/af-gateway/internal/app"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterRxServer
	service app.RxSessionManager // ИСПРАВЛЕНО: Зависим строго от абстракции интерфейса!
	log     *logger.AppLogger
}

func NewGrpcHandler(service app.RxSessionManager, log *logger.AppLogger) *GrpcHandler {
	return &GrpcHandler{service: service, log: log}
}

func (h *GrpcHandler) SendAAQuery(ctx context.Context, req *gen.MediaSessionRequest) (*gen.MediaSessionResponse, error) {
	h.log.Info("AF Rx Gateway -> Получен gRPC запрос от CDN на выделение полосы для абонента %s", req.SubscriberId)

	// Вызываем Use Case слой через интерфейс
	session, err := h.service.AuthorizeMediaSession(ctx, req.SubscriberId, "VIDEO_4K", req.DurationSeconds)
	if err != nil {
		return &gen.MediaSessionResponse{ResultCode: 5012, SessionId: ""}, nil // DIAMETER_UNABLE_TO_COMPLY
	}

	// Асинхронный таймер отмены SLA-приоритета
	time.AfterFunc(time.Until(session.ExpiresAt), func() {
		h.log.Info("AF Rx Gateway -> Время жизни сессии %s истекло. SLA приоритет отозван.", session.SessionID)
	})

	return &gen.MediaSessionResponse{
		ResultCode: 2001, // DIAMETER_SUCCESS
		SessionId:  session.SessionID,
	}, nil
}
