package app

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"pcef-shaper-system/internal/pkg/dispatch"
	"pcef-shaper-system/internal/pkg/lru" // ИМПОРТИРУЕМ ТВОЙ LRU КЭШ
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/domain"
)

const (
	StateActive      uint64 = 1 << 0
	TrafficStreaming uint64 = 1 << 1
	TrafficSocial    uint64 = 1 << 2
	PackSmall        uint64 = 1 << 3
	PackHeavy        uint64 = 1 << 4
)

type CoreShard struct {
	// Мьютекс теперь охраняет только конкурентную аллокацию внутри кэша
	mu  sync.RWMutex
	lru *lru.ReactiveLruCache // Твой реактивный кэш вместо обычной мапы!
}

type PcefCoreService struct {
	shards         []*CoreShard
	shardCount     uint32
	dispatchEngine *dispatch.TableDrivenEngine
	dpiSignatures  map[string]uint64
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

	// Финтех-колбэк: при ленивом вытеснении сессии из кэша, асинхронно возвращаем остаток кванта по Gy
	evictCallback := func(ipKey string) {
		_, _ = s.ocsClient.RequestCreditControl(context.Background(), &gen.CreditControlRequest{
			SessionId:    fmt.Sprintf("term_%s_%d", ipKey, time.Now().UnixNano()),
			SubscriberId: "250010000000001", // Для демо мапим на дефолтного юзера
			RequestType:  3,                 // TERMINATE
			UsedBytes:    0,
		})
	}

	// Внутри конструктора NewPcefCoreService, в цикле инициализации шардов:
	for i := uint32(0); i < s.shardCount; i++ {
		cacheInstance := lru.NewReactiveLruCache(5000, 60*time.Second, evictCallback)
		s.shards[i] = &CoreShard{
			lru: cacheInstance,
		}
		// Включаем часовую страховку (для демо-тестов поставим интервал 1 минуту, на проде будет 1 час)
		cacheInstance.StartHourlyJanitor(context.Background(), 1*time.Minute)
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

	sess := &domain.SubscriberSession{
		IMSI:             imsi,
		IP:               ip,
		TariffClass:      tariff,
		IsActive:         true,
		CurrentBandwidth: 100 * 1024 * 1024,
		LastHeartbeat:    time.Now(),
	}

	// Пишем через потокобезопасный метод Set твоего кэша
	s.shards[idx].lru.Set(ip, sess)
}

func (s *PcefCoreService) ProcessPacket(ctx context.Context, frame *gen.RawPacketFrame) (*gen.EnforcementVerdict, error) {
	idx := s.getShardIndex(frame.SourceIp)

	// Вытаскиваем сессию из кэша с автоматической "ленивой" проверкой протухания за O(1)
	val, exists := s.shards[idx].lru.Get(frame.SourceIp)
	if !exists {
		return &gen.EnforcementVerdict{SourceIp: frame.SourceIp, Action: "XDP_DROP"}, nil
	}
	session := val.(*domain.SubscriberSession)

	trafficBit, isKnown := s.dpiSignatures[frame.DestinationHost]
	if !isKnown {
		trafficBit = TrafficSocial
	}

	sizeBit := s.dispatchEngine.EvaluateRange(frame.PayloadSizeBytes)

	var bitmask uint64
	if session.IsActive {
		bitmask |= StateActive
	}
	bitmask |= trafficBit
	bitmask |= sizeBit

	resp, err := s.ocsClient.RequestCreditControl(ctx, &gen.CreditControlRequest{
		SessionId:    fmt.Sprintf("sess_%s_%d", session.IMSI, time.Now().UnixNano()),
		SubscriberId: session.IMSI,
		ChargingKey:  uint32(trafficBit),
		RequestType:  2,
		UsedBytes:    uint64(frame.PayloadSizeBytes),
	})

	if err != nil || resp.ResultCode == 4012 {
		return &gen.EnforcementVerdict{
			SourceIp:         frame.SourceIp,
			Action:           "THROTTLE",
			AllowedBandwidth: 64 * 1024,
		}, nil
	}

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
	s.dispatchEngine.AddRangeConfig(1024, PackSmall)
	s.dispatchEngine.AddRangeConfig(999999999999, PackHeavy)

	maskStreaming := StateActive | TrafficStreaming | PackHeavy
	s.dispatchEngine.RegisterAction(maskStreaming, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession) // Извлекаем по индексу из variadic slice
		sess.CurrentBandwidth = 50 * 1024 * 1024
		return nil
	})

	maskSocialPing := StateActive | TrafficSocial | PackSmall
	s.dispatchEngine.RegisterAction(maskSocialPing, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 100 * 1024 * 1024
		return nil
	})

	maskSocialHeavy := StateActive | TrafficSocial | PackHeavy
	s.dispatchEngine.RegisterAction(maskSocialHeavy, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 10 * 1024 * 1024
		return nil
	})
}
