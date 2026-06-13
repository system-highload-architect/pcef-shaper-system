package grpc

import (
	"context"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/ocs-rating/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterGyServer
	service app.RatingEngine
}

func NewGrpcHandler(service app.RatingEngine) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) RequestCreditControl(ctx context.Context, req *gen.CreditControlRequest) (*gen.CreditControlAnswer, error) {
	if req.SessionId == "" || req.SubscriberId == "" {
		return nil, status.Error(codes.InvalidArgument, "SessionID and SubscriberID cannot be empty")
	}

	var grantedBytes uint64
	var resultCode uint32
	var err error

	// Диспетчеризация типа запроса Gy-интерфейса согласно ТЗ
	switch req.RequestType {
	case 1: // INITIAL
		grantedBytes, resultCode, err = h.service.Reserve(ctx, req.SubscriberId, req.SessionId, req.ChargingKey, req.UsedBytes)
	case 2: // UPDATE
		grantedBytes, resultCode, err = h.service.CommitAndReserve(ctx, req.SubscriberId, req.SessionId, req.ChargingKey, req.UsedBytes, req.UsedBytes) // Используем UsedBytes как дефолтный шаг пролонгации
	case 3: // TERMINATE
		err = h.service.Release(ctx, req.SubscriberId, req.SessionId, req.ChargingKey, req.UsedBytes)
		resultCode = 2001 // SUCCESS
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported Gy RequestType: %d", req.RequestType)
	}

	if err != nil {
		// Не прерываем рантайн, возвращаем Diameter-код ошибки отсечки
		return &gen.CreditControlAnswer{
			SessionId:    req.SessionId,
			ResultCode:   4012, // DIAMETER_CREDIT_LIMIT_REACHED
			GrantedBytes: 0,
		}, nil
	}

	return &gen.CreditControlAnswer{
		SessionId:    req.SessionId,
		ResultCode:   resultCode,
		GrantedBytes: grantedBytes,
	}, nil
}
