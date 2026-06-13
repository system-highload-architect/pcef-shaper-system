package grpc

import (
	"context"

	gen "pcef-shaper-system/pb/gen" // Общие скомпилированные контракты
	"pcef-shaper-system/services/spr-storage/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedSubscriptionRepositoryServer
	service app.ProfileRepository
}

func NewGrpcHandler(service app.ProfileRepository) *GrpcHandler {
	return &GrpcHandler{service: service}
}

func (h *GrpcHandler) FetchProfile(ctx context.Context, req *gen.ProfileRequest) (*gen.SubscriberProfileResponse, error) {
	if req.Imsi == "" {
		return nil, status.Error(codes.InvalidArgument, "IMSI identifier cannot be empty")
	}

	profile, found, err := h.service.Find(ctx, req.Imsi)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Internal database failure: %v", err)
	}

	if !found {
		return nil, status.Errorf(codes.NotFound, "Subscriber with IMSI %s not found in ScyllaDB clusters", req.Imsi)
	}

	return &gen.SubscriberProfileResponse{
		Imsi:         profile.IMSI,
		TariffClass:  profile.TariffClass,
		IsSuspended:  profile.IsSuspended,
		GrantedBytes: profile.GrantedBytes,
	}, nil
}
