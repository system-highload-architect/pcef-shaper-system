package app

import (
	"context"

	"pcef-shaper-system/services/olap-analytics/internal/domain"
)

type OlapEngine interface {
	InsertBatch(ctx context.Context, records []*domain.AnalyticalRecord) error
}
