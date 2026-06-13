package grpc

import (
	"context"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/olap-analytics/internal/app"
	"pcef-shaper-system/services/olap-analytics/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedDiameterGzServer
	service app.OlapEngine
}

func NewGrpcHandler(service app.OlapEngine) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) StreamCdrLogs(ctx context.Context, req *gen.BulkCdrPack) (*gen.CdrAck, error) {
	if req.Records == nil || len(req.Records) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Batch cannot be empty")
	}

	var records []*domain.AnalyticalRecord
	for _, r := range req.Records {
		records = append(records, &domain.AnalyticalRecord{
			RecordID:     r.RecordId,
			SubscriberID: r.SubscriberId,
			BytesDumped:  r.BytesDumped,
			Timestamp:    r.Timestamp.AsTime(),
		})
	}

	if err := h.service.InsertBatch(ctx, records); err != nil {
		return nil, status.Errorf(codes.Internal, "ClickHouse Write Failure: %v", err)
	}

	return &gen.CdrAck{IsCommitted: true}, nil
}
