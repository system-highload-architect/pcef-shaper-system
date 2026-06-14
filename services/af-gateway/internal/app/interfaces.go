package app

import (
	"context"

	"pcef-shaper-system/services/af-gateway/internal/domain"
)

// RxSessionManager определяет b2b-контракт для управления прикладными медиа-сессиями по Rx
type RxSessionManager interface {
	AuthorizeMediaSession(ctx context.Context, subID string, mediaType string, duration int64) (*domain.MediaSession, error)
}
