package app

import (
	"context"
	gen "pcef-shaper-system/pb/gen"
)

// ShaperEngine описывает интерфейс обработки real-time трафика
type ShaperEngine interface {
	ProcessPacket(ctx context.Context, frame *gen.RawPacketFrame) (*gen.EnforcementVerdict, error)
	RegisterSubscriber(ctx context.Context, imsi, ip, tariff string)
}
