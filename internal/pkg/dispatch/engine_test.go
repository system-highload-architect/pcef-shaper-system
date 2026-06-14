package dispatch

import (
	"context"
	"testing"
)

// BenchmarkTableDrivenEngine замеряет скорость работы гибридного конвейера масок и бинарного поиска
// BenchmarkTableDrivenEngine evaluates the nano-speed of bitmask matching and binary search ranges
func BenchmarkTableDrivenEngine(b *testing.B) {
	engine := NewTableDrivenEngine()

	// 1. Инициализируем шкалу диапазонов (Req. 6)
	const PackSmall uint64 = 1 << 3
	const PackHeavy uint64 = 1 << 4
	engine.AddRangeConfig(1024, PackSmall)
	engine.AddRangeConfig(999999999999, PackHeavy)

	// 2. Регистрируем фейковый экшен политики на маску 0x15 (Req. 5)
	const compositeMask uint64 = (1 << 0) | (1 << 2) | PackHeavy // Active + Social + Heavy = 21 (0x15)
	engine.RegisterAction(compositeMask, func(ctx context.Context, args ...any) error {
		// Имитируем моментальное применение политики QoS в RAM
		return nil
	})

	ctx := context.Background()

	// 3. Сбрасываем таймер перед пиковым Highload-спринтом
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Имитируем горячий путь обработки пакета
		var bitmask uint64 = (1 << 0) | (1 << 2) // Active + Social

		// Бинарный поиск O(log N) по плоскому массиву интервалов
		sizeBit := engine.EvaluateRange(500 * 1024) // 500 КБ -> PackHeavy
		bitmask |= sizeBit

		// Мгновенный косвенный переход к функции по хэш-ключу маски за O(1)
		err := engine.Execute(ctx, bitmask)
		if err != nil {
			b.Fatal(err)
		}
	}
}
