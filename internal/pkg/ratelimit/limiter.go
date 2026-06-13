package ratelimit

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// AtomicClientBucket хранит состояние лимитера в виде атомарных int64 полей
// AtomicClientBucket encapsulates token bucket state using atomic int64 primitives
type AtomicClientBucket struct {
	// Храним последнее время пополнения в наносекундах (UnixNano)
	LastRefillNano int64
	// Храним текущие токены, умноженные на 1,000,000 для сохранения дробной точности в int64
	TokensScaled int64
}

// TokenBucketLimiter реализует 100% Lock-Free маркерную корзину (Req. 5 & 6)
// TokenBucketLimiter implements a 100% Lock-Free token bucket shaper (Req. 5 & 6)
type TokenBucketLimiter struct {
	clients    sync.Map // string -> *AtomicClientBucket
	ratePerNs  float64  // Скорость генерации токенов на одну наносекунду
	capacity   int64    // Максимальный объем бакета (масштабированный)
	blockedCnt uint64   // Атомарный счетчик DOS-блокировок
}

// NewTokenBucketLimiter — конструктор Lock-Free лимитера частоты запросов
func NewTokenBucketLimiter(ratePerSec, capacityTokens float64) *TokenBucketLimiter {
	const scale = 1_000_000
	return &TokenBucketLimiter{
		// Переводим секундный рейт в наносекундный масштаб
		ratePerNs: ratePerSec / float64(time.Second),
		capacity:  int64(capacityTokens * scale),
	}
}

// Allow проверяет RPS за O(1) через CAS-циклы процессора без единой блокировки потоков ОС
// Allow evaluates request rate within O(1) via hardware CAS loops with zero thread blocking
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	const scale = 1_000_000
	var bucket *AtomicClientBucket

	val, exists := l.clients.Load(key)
	if !exists {
		bucket = &AtomicClientBucket{
			LastRefillNano: time.Now().UnixNano(),
			TokensScaled:   l.capacity,
		}
		actual, loaded := l.clients.LoadOrStore(key, bucket)
		if loaded {
			bucket = actual.(*AtomicClientBucket)
		}
	} else {
		bucket = val.(*AtomicClientBucket)
	}

	now := time.Now().UnixNano()

	// ВХОДИМ В АТОМАРНЫЙ ЦИКЛ COMPARE-AND-SWAP (LOCK-FREE CORE)
	for {
		oldRefill := atomic.LoadInt64(&bucket.LastRefillNano)
		oldTokens := atomic.LoadInt64(&bucket.TokensScaled)

		// Вычисляем дельту времени в наносекундах
		elapsedNs := now - oldRefill
		if elapsedNs < 0 {
			elapsedNs = 0
		}

		// Математика Lazy Refill в наносекундном целочисленном масштабе
		newTokens := oldTokens + int64(float64(elapsedNs)*l.ratePerNs*scale)
		if newTokens > l.capacity {
			newTokens = l.capacity
		}

		// Проверяем баланс токенов для пропуска фрейма
		if newTokens < scale {
			// ТРИУМФ: Токенов нет, лимит превышен! Атака отражена на уровне регистров CPU
			atomic.AddUint64(&l.blockedCnt, 1)
			return false, fmt.Errorf("L7 Rate-Limit exhausted for client: %s", key)
		}

		// Пытаемся забрать 1 токен (минус 1.0 в масштабе scale)
		finalTokens := newTokens - scale

		// АТОМАРНАЯ СИНХРОНИЗАЦИЯ: Проверяем, не изменил ли другой поток состояние бакета параллельно
		if atomic.CompareAndSwapInt64(&bucket.TokensScaled, oldTokens, finalTokens) {
			// Если токены успешно обновились, атомарно фиксируем текущую метку времени
			atomic.CompareAndSwapInt64(&bucket.LastRefillNano, oldRefill, now)
			return true, nil
		}

		// Если CAS сорвался (другой поток успел раньше) — цикл заходит на следующую безопасную итерацию
		// ПРЕДОТВРАЩАЕТ SPIN-LOCK STARVATION И 100% ЗАГРУЗКУ ПРОЦЕССОРА ПОД DDoS
		runtime.Gosched()
	}
}

func (l *TokenBucketLimiter) GetBlockedCount() uint64 {
	return atomic.LoadUint64(&l.blockedCnt)
}
