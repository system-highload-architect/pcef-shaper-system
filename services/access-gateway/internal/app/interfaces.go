package app

import "context"

// RadiusSessionSignaling определяет контракт проброса сессий из UDP-RADIUS в Control Plane
type RadiusSessionSignaling interface {
	HandleRadiusPacket(ctx context.Context, ip string, imsi string)
}
