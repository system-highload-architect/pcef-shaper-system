package dispatch

import (
	"context"
	"fmt"
	"sort"
)

// PolicyAction определяет универсальную сигнатуру исполняемой b2b-политики
type PolicyAction func(ctx context.Context, args ...any) error

// RangeConfig связывает верхнюю границу числового диапазона с битовым флагом
type RangeConfig struct {
	UpperBound int64
	BitFlag    uint64 // Например: 1 << 4, 1 << 5
}

// TableDrivenEngine реализует гибридную битовую и интервальную диспетчеризацию (Req. 5 & Req. 6)
type TableDrivenEngine struct {
	// Плоский срез диапазонов для Cache-Locality утилизации Cache Lines процессора
	rangeBounds []RangeConfig
	// Высокопроизводительный хэшированный реестр функций O(1)
	registry map[uint64]PolicyAction
}

func NewTableDrivenEngine() *TableDrivenEngine {
	return &TableDrivenEngine{
		rangeBounds: make([]RangeConfig, 0),
		registry:    make(map[uint64]PolicyAction),
	}
}

// AddRangeConfig регистрирует интервальный диапазон. Массив строго сортируется для бинарного поиска.
func (e *TableDrivenEngine) AddRangeConfig(upperBound int64, bitFlag uint64) {
	e.rangeBounds = append(e.rangeBounds, RangeConfig{UpperBound: upperBound, BitFlag: bitFlag})
	sort.Slice(e.rangeBounds, func(i, j int) bool {
		return e.rangeBounds[i].UpperBound < e.rangeBounds[j].UpperBound
	})
}

// RegisterAction привязывает составную битовую маску условий к конкретной функции
func (e *TableDrivenEngine) RegisterAction(bitmask uint64, action PolicyAction) {
	e.registry[bitmask] = action
}

// EvaluateRange за константное время O(log N) возвращает бит, соответствующий диапазону значения
func (e *TableDrivenEngine) EvaluateRange(value int64) uint64 {
	if len(e.rangeBounds) == 0 {
		return 0
	}

	// БИНАРНЫЙ ПОИСК по плоскому массиву в CPU-кэше без Cache Misses
	idx := sort.Search(len(e.rangeBounds), func(i int) bool {
		return e.rangeBounds[i].UpperBound >= value
	})

	if idx >= len(e.rangeBounds) {
		idx = len(e.rangeBounds) - 1
	}

	return e.rangeBounds[idx].BitFlag
}

// Execute осуществляет мгновенный O(1) переход к функции по итоговой композитной битовой маске
func (e *TableDrivenEngine) Execute(ctx context.Context, bitmask uint64, args ...any) error {
	action, exists := e.registry[bitmask]
	if !exists {
		return fmt.Errorf("architecture violation: unhandled composite bitmask condition [0x%X]", bitmask)
	}

	// Атомарный косвенный переход по указателю функции
	return action(ctx, args...)
}
