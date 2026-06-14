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
	kafkaClient    gen.DiameterGzClient // ДОБАВЛЕНО: Клиент к шине логов Gz (Kafka)
}

func NewPcefCoreService(ocsClient gen.DiameterGyClient, kafkaClient gen.DiameterGzClient) *PcefCoreService {
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

	evictCallback := func(ipKey string) {
		_, _ = s.ocsClient.RequestCreditControl(context.Background(), &gen.CreditControlRequest{
			SessionId:    fmt.Sprintf("term_%s_%d", ipKey, time.Now().UnixNano()),
			SubscriberId: "250010000000001",
			RequestType:  3,
			UsedBytes:    0,
		})
	}

	for i := uint32(0); i < s.shardCount; i++ {
		s.shards[i] = &CoreShard{
			lru: lru.NewReactiveLruCache(5000, 60*time.Second, evictCallback),
		}
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

	// АСИНХРОННЫЙ КОНВЕЙЕР: Выплевываем CDR лог в Kafka, полностью освобождая горячий поток вычислений
	// ASYNC TELEMETRY PIPELINE: Offloading CDR log pushing to an independent concurrent background thread
	go s.pushCdrLog(session.IMSI, frame.PayloadSizeBytes)

	return &gen.EnforcementVerdict{
		SourceIp:         frame.SourceIp,
		Action:           "ALLOW",
		AllowedBandwidth: session.CurrentBandwidth,
	}, nil
}

func (s *PcefCoreService) pushCdrLog(imsi string, bytes int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Формируем пакет пачки и шлем в message-bus (эмулятор Kafka) по интерфейсу Gz
	_, _ = s.kafkaClient.StreamCdrLogs(ctx, &gen.BulkCdrPack{
		Records: []*gen.CallDetailRecord{
			{
				RecordId:     fmt.Sprintf("cdr_%d", time.Now().UnixNano()),
				SubscriberId: imsi,
				BytesDumped:  bytes,
				Timestamp:    timestamppb.Now(),
			},
		},
	})
}

func (s *PcefCoreService) bootstrapDispatcher() {
	s.dispatchEngine.AddRangeConfig(1024, PackSmall)
	s.dispatchEngine.AddRangeConfig(999999999999, PackHeavy)

	maskStreaming := StateActive | TrafficStreaming | PackHeavy
	s.dispatchEngine.RegisterAction(maskStreaming, func(ctx context.Context, args ...any) error {
		sess := args[0].(*domain.SubscriberSession)
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
