package app

import (
	"context"
	"hash/fnv"
	"sync"

	"pcef-shaper-system/services/spr-storage/internal/domain"
)

// ProfileShard изолирует сегмент NoSQL-памяти под локальным мьютексом
// ProfileShard isolates a NoSQL memory segment under a localized read-write mutex
type ProfileShard struct {
	mu       sync.RWMutex
	profiles map[string]*domain.SubscriberProfile
}

// StorageService реализует ProfileRepository с применением паттерна Map Sharding (Req. 4)
// StorageService implements ProfileRepository utilizing the Map Sharding pattern (Req. 4)
type StorageService struct {
	shardCount uint32
	shards     []*ProfileShard
}

// NewStorageService — конструктор шардированного эмулятора СУБД ScyllaDB
// NewStorageService constructs a sharded ScyllaDB DBMS emulator instance
func NewStorageService(shardCount uint32) *StorageService {
	s := &StorageService{
		shardCount: shardCount,
		shards:     make([]*ProfileShard, shardCount),
	}
	for i := uint32(0); i < shardCount; i++ {
		s.shards[i] = &ProfileShard{
			profiles: make(map[string]*domain.SubscriberProfile),
		}
	}
	s.bootstrapDefaultProfiles()
	return s
}

func (s *StorageService) getShardIndex(key string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32() % s.shardCount
}

// Find извлекает профиль за O(1) через неблокирующий конкурентный RLock
// Find retrieves a profile within O(1) time complexity via parallel RLock threads
func (s *StorageService) Find(ctx context.Context, imsi string) (*domain.SubscriberProfile, bool, error) {
	shardIdx := s.getShardIndex(imsi)
	shard := s.shards[shardIdx]

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	profile, exists := shard.profiles[imsi]
	return profile, exists, nil
}

// Store атомарно сохраняет профиль в изолированный бакет памяти (Req. 4)
// Store atomically writes a profile into an isolated memory bucket (Req. 4)
func (s *StorageService) Store(ctx context.Context, profile *domain.SubscriberProfile) error {
	shardIdx := s.getShardIndex(profile.IMSI)
	shard := s.shards[shardIdx]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.profiles[profile.IMSI] = profile
	return nil
}

// Наполняем базу валидными b2b-данными для тестов согласно нашему ТЗ
// Bootstrapping the database with valid b2b datasets for telemetry verification
func (s *StorageService) bootstrapDefaultProfiles() {
	defaults := []*domain.SubscriberProfile{
		{IMSI: "250010000000001", TariffClass: "VIP", IsSuspended: false, GrantedBytes: 500 * 1024 * 1024}, // 500 МБ
		{IMSI: "250010000000002", TariffClass: "BASE", IsSuspended: false, GrantedBytes: 50 * 1024 * 1024}, // 50 МБ
		{IMSI: "250010000000003", TariffClass: "BASE", IsSuspended: true, GrantedBytes: 0},                 // Блокирован
	}

	for _, p := range defaults {
		shardIdx := s.getShardIndex(p.IMSI)
		s.shards[shardIdx].profiles[p.IMSI] = p
	}
}
