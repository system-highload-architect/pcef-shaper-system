package app

import (
	"context"

	"pcef-shaper-system/services/spr-storage/internal/domain"
)

// ProfileRepository определяет контракт для высоконагруженных NoSQL операций с профилями
// ProfileRepository defines the contract for high-throughput NoSQL operations on profiles
type ProfileRepository interface {
	Find(ctx context.Context, imsi string) (*domain.SubscriberProfile, bool, error)
	Store(ctx context.Context, profile *domain.SubscriberProfile) error
}
