package domain

import "time"

// MediaSession инкапсулирует 3GPP Rx параметры сессии контент-провайдера (IPTV/CDN)
type MediaSession struct {
	SessionID    string    `json:"session_id"`
	SubscriberID string    `json:"subscriber_id"`
	RequiredMbps int64     `json:"required_mbps"` // Запрашиваемая полоса пропускания
	MediaType    string    `json:"media_type"`    // "VIDEO_4K", "IPTV", "VOICE"
	ExpiresAt    time.Time `json:"expires_at"`
}
