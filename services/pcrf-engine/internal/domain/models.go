package domain

// PolicyProfile описывает скомпилированные правила для абонента
type PolicyProfile struct {
	IMSI        string
	TariffClass string // "VIP", "BASE", "IOT"
	RuleNames   []string
}
