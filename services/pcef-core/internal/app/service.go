package app

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"pcef-shaper-system/internal/pkg/dispatch"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/domain"
)

// Битовые константы составных условий (Req. 6)
const (
	StateActive      uint64 = 1 << 0 // Абонент активен
	TrafficStreaming uint64 = 1 << 1 // Тяжелый медиа-трафик
	TrafficSocial    uint64 = 1 << 2 // Соцсети/Мессенджеры
	PackSmall        uint64 = 1 << 3 // Сигнализация (до 1 КБ)
	PackHeavy        uint64 = 1 << 4 // Тяжелый пакет фрейма
)

type CoreShard struct {
	mu       sync.RWMutex
	sessions map[string]*domain.SubscriberSession
}

type PcefCoreService struct {
	shards         []*CoreShard
	shardCount     uint32
	dispatchEngine *dispatch.TableDrivenEngine
	dpiSignatures  map[string]uint64 // Карта SNI -> Бит типа трафика
	ocsClient      gen.DiameterGyClient
}

func NewPcefCoreService(ocsClient gen.DiameterGyClient) *PcefCoreService {
	s := &PcefCoreService{
		shardCount:     32,
		shards:         make([]*CoreShard, 32),
		dispatchEngine: dispatch.NewTableDrivenEngine(),
		ocsClient:      ocsClient,
		dpiSignatures: map[string]uint64{
			"youtube.com":  TrafficStreaming,
			"netflix.com":  TrafficStreaming,
			"telegram.org": TrafficSocial,
			"whatsapp.com": TrafficSocial,
		},
	}

	for i := uint32(0); i < s.shardCount; i++ {
		s.shards[i] = &CoreShard{sessions: make(map[string]*domain.SubscriberSession)}
	}

	s.bootstrapDispatcher()
	return s
}

func (s *PcefCoreService) getShardIndex(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32() % s.shardCount
}

func (s *PcefCoreService) RegisterSubscriber(ctx context.Context, imsi, ip, tariff string) {
	idx := s.getShardIndex(ip)
	s.shards[idx].mu.Lock()
	defer s.shards[idx].mu.Unlock()

	s.shards[idx].sessions[ip] = &domain.SubscriberSession{
		IMSI:             imsi,
		IP:               ip,
		TariffClass:      tariff,
		IsActive:         true,
		CurrentBandwidth: 100 * 1024 * 1024, // По умолчанию 100 Mbps
		LastHeartbeat:    time.Now(),
	}
}

// ProcessPacket — пиковый конвейер обработки L4-L7 фреймов без if-else каскадов
func (s *PcefCoreService) ProcessPacket(ctx context.Context, frame *gen.RawPacketFrame) (*gen.EnforcementVerdict, error) {
	// 1. Извлекаем сессию из шардированного кэша по IP за O(1) (Req. 4)
	idx := s.getShardIndex(frame.SourceIp)
	s.shards[idx].mu.RLock()
	session, exists := s.shards[idx].sessions[frame.SourceIp]
	s.shards[idx].mu.RUnlock()

	if !exists {
		return &gen.EnforcementVerdict{SourceIp: frame.SourceIp, Action: "XDP_DROP"}, nil
	}

	// 2. Встроенный DPI-анализатор сигнатур хоста за O(1) (Req. 1)
	trafficBit, isKnown := s.dpiSignatures[frame.DestinationHost]
	if !isKnown {
		trafficBit = TrafficSocial // Дефолтный fallback класс
	}

	// 3. Вычисление интервального диапазона пакета через Бинарный Поиск за O(log N) (Req. 6)
	sizeBit := s.dispatchEngine.EvaluateRange(frame.PayloadSizeBytes)

	// 4. Сборка композитной битовой маски оператором ИЛИ (Req. 6)
	var bitmask uint64
	if session.IsActive {
		bitmask |= StateActive
	}
	bitmask |= trafficBit
	bitmask |= sizeBit

	// 5. Онлайн-квитирование по Gy Diameter интерфейсу (Req. 2)
	// Перед пропуском пакета идем в ocs-rating за квантом
	resp, err := s.ocsClient.RequestCreditControl(ctx, &gen.CreditControlRequest{
		SessionId:    fmt.Sprintf("sess_%s_%d", session.IMSI, time.Now().UnixNano()),
		SubscriberId: session.IMSI,
		ChargingKey:  uint32(trafficBit),
		RequestType:  2, // UPDATE/Квантование
		UsedBytes:    uint64(frame.PayloadSizeBytes),
	})

	// Пограничное условие: Баланс исчерпан (Код отсечки 4012)
	if err != nil || resp.ResultCode == 4012 {
		return &gen.EnforcementVerdict{
			SourceIp:         frame.SourceIp,
			Action:           "THROTTLE",
			AllowedBandwidth: 64 * 1024, // Жесткий шейпинг до 64 Кбит/с по ТЗ
		}, nil
	}

	// 6. Табличная диспетчеризация за O(1) без if-else ветвлений (Req. 5)
	// Передаем маску в наш общий движок, который выполнит привязанную функцию
	err = s.dispatchEngine.Execute(ctx, bitmask, session, frame)
	if err != nil {
		return nil, err
	}

	return &gen.EnforcementVerdict{
		SourceIp:         frame.SourceIp,
		Action:           "ALLOW",
		AllowedBandwidth: session.CurrentBandwidth,
	}, nil
}

func (s *PcefCoreService) bootstrapDispatcher() {
	// Конфигурируем шкалу интервалов пакетов для бинарного поиска
	s.dispatchEngine.AddRangeConfig(1024, PackSmall)         // Сигнализация до 1 КБ
	s.dispatchEngine.AddRangeConfig(999999999999, PackHeavy) // Тяжелый контент

	// Кэшируем комбинации условий в хэш-таблицу функций
	// Комбинация: Активен + Видео-Стриминг + Тяжелый Пакет фрейма
	maskStreaming := StateActive | TrafficStreaming | PackHeavy
	s.dispatchEngine.RegisterAction(maskStreaming, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 50 * 1024 * 1024 // Выделяем 50 Mbps под YouTube
		sess.QosClassIdentifier = 6
		return nil
	})

	// Комбинация: Активен + Мессенджеры + Маленький Пакет (Пинг)
	maskSocialPing := StateActive | TrafficSocial | PackSmall
	s.dispatchEngine.RegisterAction(maskSocialPing, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 100 * 1024 * 1024 // Максимальная скорость для пингов
		sess.QosClassIdentifier = 1               // Наивысший Low-Latency приоритет 3GPP
		return nil
	})
}
