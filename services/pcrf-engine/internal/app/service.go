package app

import (
	"context"
	"fmt"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcrf-engine/internal/domain"
)

type PcrfService struct {
	sprClient  gen.SubscriptionRepositoryClient
	pcefClient gen.DiameterGxClient // ИСПРАВЛЕНО: Строго привязываем к интерфейсу Gx клиента!
	rulesMap   map[string][]string
}

func NewPcrfService(spr gen.SubscriptionRepositoryClient, pcef gen.DiameterGxClient) *PcrfService {
	return &PcrfService{
		sprClient:  spr,
		pcefClient: pcef,
		rulesMap: map[string][]string{
			"VIP":  {"VIP_UNLIMITED"},
			"BASE": {"BASE_TARIFF"},
		},
	}
}

// ... (весь остальной метод CompileRules остается без изменений)

func (s *PcrfService) CompileRules(ctx context.Context, imsi string, ipAddr string) (*domain.PolicyProfile, error) {
	// 1. Извлекаем профиль из ScyllaDB (spr-storage)
	profile, err := s.sprClient.FetchProfile(ctx, &gen.ProfileRequest{Imsi: imsi})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile from SPR: %w", err)
	}

	rules := s.rulesMap[profile.TariffClass]
	if profile.IsSuspended {
		rules = []string{"QUOTA_EXHAUSTED"}
	}

	// 2. АТОМАРНАЯ СИНХРОНИЗАЦИЯ: Раскатываем правила в RAM исполнительного ядра pcef-core по gRPC!
	// (Для демо повторно используем этот же контракт для инъекции сессии в ядро)
	_, err = s.pcefClient.ProvisionPccRules(ctx, &gen.PccRulesProvision{
		Imsi:            imsi,
		IpAddress:       ipAddr,
		ActiveRuleNames: rules,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to push PCC-rules down to PCEF-Core User Plane: %w", err)
	}

	return &domain.PolicyProfile{
		IMSI:        imsi,
		TariffClass: profile.TariffClass,
		RuleNames:   rules,
	}, nil
}
