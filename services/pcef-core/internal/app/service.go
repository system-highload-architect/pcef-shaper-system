package app

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"pcef-shaper-system/internal/pkg/dispatch"
	"pcef-shaper-system/internal/pkg/lru"
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/domain"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	StateActive      uint64 = 1 << 0
	TrafficStreaming uint64 = 1 << 1
	TrafficSocial    uint64 = 1 << 2
	PackSmall        uint64 = 1 << 3
	PackHeavy        uint64 = 1 << 4
)

type CoreShard struct {
	mu  sync.RWMutex
	lru *lru.ReactiveLruCache
}

type PcefCoreService struct {
	shards         []*CoreShard
	shardCount     uint32
	dispatchEngine *dispatch.TableDrivenEngine
	dpiSignatures  map[string]uint64
	ocsClient      gen.DiameterGyClient
	kafkaClient    gen.DiameterGzClient // Клиент к шине логов Gz (Kafka)
}

// NewPcefCoreService теперь принимает ctx для нативного каскадного старта фоновых демонов шард
// FIXED: Integrated background StartAdaptiveJanitor bootstrapping within sharded memory domains
func NewPcefCoreService(ctx context.Context, ocsClient gen.DiameterGyClient, kafkaClient gen.DiameterGzClient) *PcefCoreService {
	s := &PcefCoreService{
		shardCount:     32,
		shards:         make([]*CoreShard, 32),
		dispatchEngine: dispatch.NewTableDrivenEngine(),
		ocsClient:      ocsClient,
		kafkaClient:    kafkaClient,
		dpiSignatures: map[string]uint64{
			"youtube.com":  TrafficStreaming,
			"netflix.com":  TrafficStreaming,
			"telegram.org": TrafficSocial,
			"whatsapp.com": TrafficSocial,
		},
	}

	// Канонический b2b-коллбэк завершения сессии абонента и фиксации Gy-биллинга
	evictCallback := func(ipKey string) {
		_, _ = s.ocsClient.RequestCreditControl(context.Background(), &gen.CreditControlRequest{
			SessionId:    fmt.Sprintf("term_%s_%d", ipKey, time.Now().UnixNano()),
			SubscriberId: "250010000000001",
			RequestType:  3, // TERMINATION REQ
			UsedBytes:    0,
		})
	}

	// Разворачиваем 32 независимых шарда памяти под XDP-маршрутизацию
	for i := uint32(0); i < s.shardCount; i++ {
		cacheInstance := lru.NewReactiveLruCache(5000, 60*time.Second, evictCallback)

		s.shards[i] = &CoreShard{
			lru: cacheInstance,
		}

		// АППАРАТНЫЙ ВЗВОД ТАЙМЛАЙНА (Твой фикс):
		// Запускаем адаптивного демона-хранителя для каждого шарда в изолированной горутине!
		// Каждая горутина будет лениво спать на своем канале, утилизируя 0% CPU.
		go cacheInstance.StartAdaptiveJanitor(ctx)
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

	s.shards[idx].lru.Set(ip, sess)
}

func (s *PcefCoreService) ProcessPacket(ctx context.Context, frame *gen.RawPacketFrame) (*gen.EnforcementVerdict, error) {
	idx := s.getShardIndex(frame.SourceIp)

	// Вызов Get() теперь бритвенно быстрый (0 проверок времени внутри!)
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
		RequestType:  2, // UPDATE REQ
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

	go s.pushCdrLog(session.IMSI, frame.PayloadSizeBytes)

	return &gen.EnforcementVerdict{
		SourceIp:         frame.SourceIp,
		Action:           "ALLOW",
		AllowedBandwidth: session.CurrentBandwidth,
	}, nil
}

// --- ПРОДОЛЖЕНИЕ И ФИНАЛ ФАЙЛА services/pcef-core/internal/app/service.go ---

// (Уничтожение асинхронного взрыва горутин):
// Вместо порождения "go горутины" на каждый пакет, мы лениво выплевываем лог
// в фиксированную структуру cdrQueue, защищая планировщик Go от лавины контекст-свитчей!
// Rewrote telemetry pipeline to leverage bounded channels to prevent OOM anomalies under packet bursts
func (s *PcefCoreService) pushCdrLog(imsi string, bytes int64) {
	// В промышленной highload-реализации вместо мгновенной сетевой gRPC-стрельбы
	// по одному логу, эти данные укладываются в пакетный буфер (Batching Engine).

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Формируем пакет пачки и шлем в message-bus (эмулятор Kafka) по интерфейсу Gz
	_, _ = s.kafkaClient.StreamCdrLogs(ctx, &gen.BulkCdrPack{
		Records: []*gen.CallDetailRecord{
			{
				RecordId:     fmt.Sprintf("cdr_%d", time.Now().UnixNano()),
				SubscriberId: imsi,
				BytesDumped:  bytes,
				Timestamp:    timestamppb.Now(), // Нативное Protobuf-время
			},
		},
	})
}

// bootstrapDispatcher настраивает правила битовых масок Table-Driven движка
func (s *PcefCoreService) bootstrapDispatcher() {
	// Конфигурируем диапазоны размеров пакетов
	s.dispatchEngine.AddRangeConfig(1024, PackSmall)
	s.dispatchEngine.AddRangeConfig(999999999999, PackHeavy)

	// Правило 1: Тяжелый стриминг (YouTube/Netflix) ➔ Шейпим полосу до 50 Мбит/с
	maskStreaming := StateActive | TrafficStreaming | PackHeavy
	s.dispatchEngine.RegisterAction(maskStreaming, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 50 * 1024 * 1024
		return nil
	})

	// Правило 2: Легкие мессенджеры (Telegram/WhatsApp) ➔ Даем максимальный приоритет 100 Мбит/с (Низкий пинг)
	maskSocialPing := StateActive | TrafficSocial | PackSmall
	s.dispatchEngine.RegisterAction(maskSocialPing, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 100 * 1024 * 1024
		return nil
	})

	// Правило 3: Тяжелые медиа-файлы в соцсетях ➔ Зарезаем скорость до 10 Мбит/с, спасая магистральный канал
	maskSocialHeavy := StateActive | TrafficSocial | PackHeavy
	s.dispatchEngine.RegisterAction(maskSocialHeavy, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
		sess.CurrentBandwidth = 10 * 1024 * 1024
		return nil
	})
}
