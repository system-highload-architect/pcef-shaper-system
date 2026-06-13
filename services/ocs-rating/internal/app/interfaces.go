package app

import (
	"context"
)

// RatingEngine определяет контракт для высоконагруженного квантования по интерфейсу Gy
// RatingEngine defines the contract for high-throughput Gy interface quantization
type RatingEngine interface {
	Reserve(ctx context.Context, subID string, sessionID string, chargingKey uint32, requestedBytes uint64) (uint64, uint32, error)
	CommitAndReserve(ctx context.Context, subID string, sessionID string, chargingKey uint32, usedBytes uint64, requestedBytes uint64) (uint64, uint32, error)
	Release(ctx context.Context, subID string, sessionID string, chargingKey uint32, usedBytes uint64) error
}
