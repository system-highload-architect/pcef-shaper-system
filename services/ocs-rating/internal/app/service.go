package app

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"pcef-shaper-system/services/ocs-rating/internal/domain"
)

type OcsService struct {
	// Карта балансов абонентов. Поиск за O(1).
	balances sync.Map // string -> *domain.GyCreditState
	// Карта активных Gy-сессий
	sessions sync.Map // string -> *domain.OcsSession
}

func NewOcsService() *OcsService {
	s := &OcsService{}
	s.bootstrapDefaultBalances()
	return s
}

// 1. Reserve (INITIAL Request) — бронирование первого кванта трафика
func (s *OcsService) Reserve(ctx context.Context, subID string, sessionID string, chargingKey uint32, requestedBytes uint64) (uint64, uint32, error) {
	val, exists := s.balances.Load(subID)
	if !exists {
		return 0, 4012, fmt.Errorf("subscriber profile missing in OCS core") // DIAMETER_USER_UNKNOWN
	}
	state := val.(*domain.GyCreditState)

	for {
		total := atomic.LoadUint64(&state.TotalBalanceBytes)
		if total == 0 {
			return 0, 4012, nil // DIAMETER_CREDIT_LIMIT_REACHED (ТЗ: Баланс 0 -> Код отсечки)
		}

		// Вычисляем размер кванта (если денег меньше, чем просили — отдаем остаток)
		granted := requestedBytes
		if total < requestedBytes {
			granted = total
		}

		// Lock-Free модификация через CAS (Compare-And-Swap) цикл
		if atomic.CompareAndSwapUint64(&state.TotalBalanceBytes, total, total-granted) {
			atomic.AddUint64(&state.ReservedBytes, granted)

			// Фиксируем сессию
			s.sessions.Store(sessionID, &domain.OcsSession{
				SessionID:    sessionID,
				SubscriberID: subID,
				ChargingKey:  chargingKey,
				CurrentQuota: granted,
				LastUpdate:   time.Now(),
			})
			return granted, 2001, nil // DIAMETER_SUCCESS
		}
	}
}

// 2. CommitAndReserve (UPDATE Request) — окончательное списание старого кванта и бронь нового
func (s *OcsService) CommitAndReserve(ctx context.Context, subID string, sessionID string, chargingKey uint32, usedBytes uint64, requestedBytes uint64) (uint64, uint32, error) {
	val, exists := s.balances.Load(subID)
	if !exists {
		return 0, 4012, fmt.Errorf("subscriber profile missing")
	}
	state := val.(*domain.GyCreditState)

	// Списываем использованные байты из резерва
	for {
		reserved := atomic.LoadUint64(&state.ReservedBytes)
		finalUsed := usedBytes
		if reserved < usedBytes {
			finalUsed = reserved
		}
		if atomic.CompareAndSwapUint64(&state.ReservedBytes, reserved, reserved-finalUsed) {
			break
		}
	}

	// Запрашиваем следующий квант для пролонгации сессии
	return s.Reserve(ctx, subID, sessionID, chargingKey, requestedBytes)
}

// 3. Release (TERMINATE Request) — списание потраченного и атомарный возврат неиспользованного остатка
func (s *OcsService) Release(ctx context.Context, subID string, sessionID string, chargingKey uint32, usedBytes uint64) error {
	val, exists := s.balances.Load(subID)
	if !exists {
		return fmt.Errorf("subscriber profile missing")
	}
	state := val.(*domain.GyCreditState)

	sessVal, sessExists := s.sessions.Load(sessionID)
	if !sessExists {
		return fmt.Errorf("active session context not found")
	}
	session := sessVal.(*domain.OcsSession)

	// Вычисляем остаток, который абонент забронировал, но не истратил
	var refund uint64
	if session.CurrentQuota > usedBytes {
		refund = session.CurrentQuota - usedBytes
	}

	// Возвращаем остаток кванта обратно на основной счет абонента
	atomic.AddUint64(&state.TotalBalanceBytes, refund)

	// Очищаем резерв
	for {
		reserved := atomic.LoadUint64(&state.ReservedBytes)
		decrement := session.CurrentQuota
		if reserved < decrement {
			decrement = reserved
		}
		if atomic.CompareAndSwapUint64(&state.ReservedBytes, reserved, reserved-decrement) {
			break
		}
	}

	s.sessions.Delete(sessionID)
	return nil
}

func (s *OcsService) bootstrapDefaultBalances() {
	// Добавляем емкости: VIP даем 100 Гигабайт, BASE — 5 Гигабайт
	s.balances.Store("250010000000001", &domain.GyCreditState{TotalBalanceBytes: 100 * 1024 * 1024 * 1024})
	s.balances.Store("250010000000002", &domain.GyCreditState{TotalBalanceBytes: 5 * 1024 * 1024 * 1024})
	s.balances.Store("250010000000003", &domain.GyCreditState{TotalBalanceBytes: 0})
}
