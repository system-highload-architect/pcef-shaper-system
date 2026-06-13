package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ClientBucket struct {
	mu         sync.Mutex
	lastRefill time.Time
	tokens     float64
}

// TokenBucketLimiter защищает L7-вход микросервиса от флуда
type TokenBucketLimiter struct {
	clients  sync.Map // string -> *ClientBucket
	rate     float64  // Токенов в секунду
	capacity float64  // Максимальный всплеск (Burst)
	blocked  uint64   // Атомарный счетчик блокировок
}

func NewTokenBucketLimiter(rate, capacity float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:     rate,
		capacity: capacity,
	}
}

// Allow проверяет частоту запросов идентификатора (IP/IMSI) за O(1) без глобальных локов
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	var bucket *ClientBucket
	val, exists := l.clients.Load(key)

	if !exists {
		bucket = &ClientBucket{
			lastRefill: time.Now(),
			tokens:     l.capacity,
		}
		actual, loaded := l.clients.LoadOrStore(key, bucket)
		if loaded {
			bucket = actual.(*ClientBucket)
		}
	} else {
		bucket = val.(*ClientBucket)
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.lastRefill = now

	// Lazy Refill математика
	bucket.tokens += elapsed * l.rate
	if bucket.tokens > l.capacity {
		bucket.tokens = l.capacity
	}

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true, nil
	}

	atomic.AddUint64(&l.blocked, 1)
	return false, fmt.Errorf("rate limit exceeded for key %s", key)
}

func (l *TokenBucketLimiter) GetBlockedCount() uint64 {
	return atomic.LoadUint64(&l.blocked)
}
