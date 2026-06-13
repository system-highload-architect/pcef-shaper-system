package domain

import "time"

// SimulatedDevice инкапсулирует параметры эмулируемого смартфона абонента
type SimulatedDevice struct {
	IMSI        string
	IP          string
	TargetHosts []string      // Пулы сайтов, куда юзер «ходит» (youtube.com, telegram.org)
	PacingDelay time.Duration // Задержка джиттера между отправками пакетов
}
