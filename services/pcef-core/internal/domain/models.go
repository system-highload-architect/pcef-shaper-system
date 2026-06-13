package domain

import "time"

// SubscriberSession инкапсулирует динамическое QoS-состояние абонента в RAM ядра
type SubscriberSession struct {
	IMSI               string
	IP                 string
	TariffClass        string // "VIP", "BASE", "IOT"
	IsActive           bool
	CurrentBandwidth   int64  // Выделенная скорость в битах/сек (Шейпинг)
	QosClassIdentifier uint32 // Приоритет пакета QCI (3GPP)
	LastHeartbeat      time.Time
}
