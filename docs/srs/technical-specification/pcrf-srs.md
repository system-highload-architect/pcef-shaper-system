# ⚙️ Policy & Charging Rules Function (PCRF) — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Централизованная оркестрация бизнес-логики телеком/финтех сети. Сборка PCC-правил из сырых профилей ScyllaDB и отправка их в User Plane за константное время $O(1)$.
* **[EN]** Centralized orchestration of telecom/fintech network business logic. Compiling PCC rules from raw ScyllaDB profiles and dispatching them into the User Plane within constant $O(1)$ time complexity.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type PccRule struct {
	RuleName         string `json:"rule_name"`          // Уникальный ID правила: "YouTube_Premium"
	ChargingKey      uint32 `json:"charging_key"`       // Ключ тарификации OCS: 10 (Free), 20 (Pay)
	MaxRatingUplink  int64  `json:"max_rating_uplink"`  // Лимит отдачи в битах/сек
	MaxRatingDownlink int64  `json:"max_rating_downlink"` // Лимит скачивания в битах/сек
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** При обработке запроса `CCR-INITIAL`, PCRF атомарно извлекает профиль абонента из SPR. Логика композитных ключей: если тариф равен `VIP`, а статус `ACTIVE`, PCRF на лету подставляет готовый указатель из хэш-таблицы правил, полностью минуя `if-else` [🧠]. Если профиль заблокирован, генерируется правило со скоростью Downlink = 0 [🧠].
* **[EN]** Upon `CCR-INITIAL` evaluation, PCRF atomically fetches the subscriber profile from SPR. Composite key logic: if tariff equals `VIP` and state is `ACTIVE`, PCRF instantly fetches the pre-cached function pointer, evading nested condition evaluations. If the profile is suspended, it returns a rule chocked to 0 Downlink speed.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Полное отсутствие условных ветвлений `if-else` и `switch` на горячих путях компиляции правил [🧠].
2. Устойчивость к `Branch Misprediction` за счет перевода логики на `map[string]*PccRule` [🧠].
