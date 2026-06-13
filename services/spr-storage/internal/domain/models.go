package domain

// SubscriberProfile представляет b2b-паспорт контракта абонента в NoSQL хранилище
// SubscriberProfile represents the b2b subscriber contract passport in NoSQL storage
type SubscriberProfile struct {
	IMSI         string `json:"imsi"`          // Уникальный идентификатор SIM-карты / Unique SIM identifier
	TariffClass  string `json:"tariff_class"`  // Класс тарифа: "VIP", "BASE", "IOT"
	IsSuspended  bool   `json:"is_suspended"`  // Флаг принудительной блокировки / Admin suspension flag
	GrantedBytes int64  `json:"granted_bytes"` // Доступный лимит трафика в байтах / Available data volume in bytes
}
