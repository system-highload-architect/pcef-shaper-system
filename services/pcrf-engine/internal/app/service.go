package app

import (
	"context"
	"fmt"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcrf-engine/internal/domain"
)

type PcrfService struct {
	sprClient gen.SubscriptionRepositoryClient
	rulesMap  map[string][]string // Табличный реестр тарифов O(1) без if-else
}

func NewPcrfService(sprClient gen.SubscriptionRepositoryClient) *PcrfService {
	return &PcrfService{
		sprClient: sprClient,
		rulesMap: map[string][]string{
			"VIP":  {"YouTube_Premium", "Social_Unlim", "Gaming_LowLatency"},
			"BASE": {"Social_Unlim"},
			"IOT":  {"Telemetry_Only"},
		},
	}
}

func (s *PcrfService) CompileRules(ctx context.Context, imsi string) (*domain.PolicyProfile, error) {
	// 1. Запрашиваем сырой паспорт абонента из NoSQL ScyllaDB по интерфейсу Sp
	profile, err := s.sprClient.FetchProfile(ctx, &gen.ProfileRequest{Imsi: imsi})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile from SPR layer: %w", err)
	}

	// Пограничное условие: абонент заблокирован — выставляем пустые правила (0 скорости)
	if profile.IsSuspended {
		return &domain.PolicyProfile{IMSI: imsi, TariffClass: profile.TariffClass, RuleNames: []string{}}, nil
	}

	// 2. Табличный выбор правил за O(1) без if-else
	rules, exists := s.rulesMap[profile.TariffClass]
	if !exists {
		rules = []string{"Generic_Blocked"}
	}

	return &domain.PolicyProfile{
		IMSI:        imsi,
		TariffClass: profile.TariffClass,
		RuleNames:   rules,
	}, nil
}
