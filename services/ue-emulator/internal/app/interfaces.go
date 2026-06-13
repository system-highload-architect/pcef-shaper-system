package app

import "context"

// GeneratorEngine описывает интерфейс запуска Highload стресс-нагрузки
type GeneratorEngine interface {
	StartLoadTest(ctx context.Context) error
	StopLoadTest()
}
