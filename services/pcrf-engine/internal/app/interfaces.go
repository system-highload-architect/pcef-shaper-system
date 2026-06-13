package app

import (
	"context"

	"pcef-shaper-system/services/pcrf-engine/internal/domain"
)

type PolicyEngine interface {
	CompileRules(ctx context.Context, imsi string) (*domain.PolicyProfile, error)
}
