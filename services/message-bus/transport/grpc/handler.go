package grpc

import (
	"context"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/message-bus/internal/app"
	"pcef-shaper-system/services/message-bus/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterGzServer
	service app.MessageBroker
}

func NewGrpcHandler(service app.MessageBroker) *GrpcHandler {
	return &GrpcHandler{service: service}
}

// StreamCdrLogs принимает пачки асинхронных логов от ядра и пушит их в каналы Kafka
func (h *GrpcHandler) StreamCdrLogs(ctx context.Context, req *gen.BulkCdrPack) (*gen.CdrAck, error) {
	if req.Records == nil || len(req.Records) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Bulk pack records cannot be empty")
	}

	for _, record := range req.Records {
		event := &domain.CdrEvent{
			RecordID:     record.RecordId,
			SubscriberID: record.SubscriberId,
			BytesDumped:  record.BytesDumped,
			Timestamp:    record.Timestamp.AsTime(),
		}

		// Асинхронно пушим в очередь
		_ = h.service.Publish(ctx, event)
	}

	return &gen.CdrAck{IsCommitted: true}, nil
}
