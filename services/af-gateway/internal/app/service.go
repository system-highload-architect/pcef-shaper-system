package app

import (
	"context"
	"fmt"
	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/services/af-gateway/internal/domain"
	"time"
)

type AfService struct {
	log *logger.AppLogger
}

func NewAfService(log *logger.AppLogger) *AfService {
	return &AfService{log: log}
}

// AuthorizeMediaSession реализует интерфейс RxSessionManager
func (s *AfService) AuthorizeMediaSession(ctx context.Context, subID string, mediaType string, duration int64) (*domain.MediaSession, error) {
	sessionID := fmt.Sprintf("rx_session_cdn_%d", time.Now().UnixNano())

	session := &domain.MediaSession{
		SessionID:    sessionID,
		SubscriberID: subID,
		RequiredMbps: 50,
		MediaType:    mediaType,
		ExpiresAt:    time.Now().Add(time.Duration(duration) * time.Second), // time.Duration нативно перемножится с int64
	}

	s.log.Info("AF Service -> Создана доменная медиа-сессия [%s] для абонента [%s]", sessionID, subID)
	return session, nil
}
