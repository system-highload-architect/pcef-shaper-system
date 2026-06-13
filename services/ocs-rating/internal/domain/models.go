package domain

import "time"

// GyCreditState отражает балансы квот конкретного абонента в RAM
// GyCreditState represents subscriber quota balances within RAM indices
type GyCreditState struct {
	TotalBalanceBytes uint64 // Общий доступный счетчик байт / Global counter of available bytes
	ReservedBytes     uint64 // Замороженный квант в текущей сессии / Frozen volume chunk within active session
}

// OcsSession инкапсулирует Diameter Gy контекст сессии на стороне OCS
// OcsSession encapsulates the Diameter Gy session context on the OCS end
type OcsSession struct {
	SessionID    string
	SubscriberID string
	ChargingKey  uint32
	CurrentQuota uint64
	LastUpdate   time.Time
}
