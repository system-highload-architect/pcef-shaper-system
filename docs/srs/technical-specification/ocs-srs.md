# 💸 Online Charging System (OCS) — Technical Requirements Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Эмуляция распределенного высоконагруженного финтех-ядра реального времени (на базе архитектуры Aerospike HMA) для атомарного резервирования, списания и возврата квот трафика (мегабайт) абонентов по интерфейсу Gy Diameter без Mutex Contention.
* **[EN]** Emulation of a highly concurrent distributed real-time fintech core (powered by Aerospike HMA architecture) designed for atomic reservation, deduction, and refund of subscriber data quotas via the Diameter Gy interface with zero Mutex Contention.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
// GyCreditState отражает балансы квот абонента в RAM-индексах Aerospike
// GyCreditState represents subscriber quota balances within RAM indices
type GyCreditState struct {
	TotalBalanceBytes uint64 // Общий доступный счетчик байт / Global counter of available bytes
	ReservedBytes     uint64 // Замороженный квант в текущей сессии / Frozen volume chunk within active session
}

// OcsClientSession инкапсулирует Diameter Gy контекст сессии на стороне OCS
// OcsClientSession encapsulates the Diameter Gy session context on the OCS end
type OcsClientSession struct {
	SessionID    string
	SubscriberID string
	ChargingKey  uint32
	CurrentQuota uint64 // Размер текущего выданного кванта (ТЗ: 10 МБ) / Size of currently granted chunk (SRS: 10MB)
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия

* **[RU]** **Паттерн Квантования (Quota Reservation Physics):**
  * `INITIAL Request (Запрос первого кванта):` При старте сессии OCS проверяет `TotalBalanceBytes`. Если баланс $\ge 10$ МБ (`10*1024*1024` байт), OCS атомарно вычитает 10 МБ из общего баланса, переносит их в `ReservedBytes` и возвращает в PCEF код `DIAMETER_SUCCESS (2001)` с дескриптором `Granted-Service-Unit` [🧠].
  * `UPDATE Request (Списание и пролонгация):` Когда абонент выкачал квант, PCEF шлет отчет. OCS берет 10 МБ из `ReservedBytes` и окончательно списывает их в утиль, после чего пытается забронировать следующие 10 МБ [🧠].
  * `TERMINATE Request (Возврат остатка):` Если сессия закрылась, а абонент выкачал только 4 МБ из 10, PCEF возвращает неиспользованные 6 МБ [🧠]. OCS списывает 4 МБ, а оставшиеся 6 МБ **атомарно возвращает обратно** в `TotalBalanceBytes`, защищая деньги пользователя [🧠].
* **[RU]** **Пограничные условия (Boundary Conditions):**
  * Если `TotalBalanceBytes` меньше 10 МБ, но больше 0, OCS выдает «остаточный квант» (ровно столько байт, сколько осталось на балансе) [🧠].
  * Если баланс равен 0, OCS возвращает финтех-код отсечки `DIAMETER_CREDIT_LIMIT_REACHED (4012)`, заставляя PCEF Core мгновенно запустить QoS-шейпинг до 64 Кбит/с [🧠].

* **[EN]** **Quota Reservation Physics & Logic:**
  * `INITIAL Request:` Upon session birth, OCS evaluates `TotalBalanceBytes`. If balance $\ge 10$ MB, OCS atomically subtracts 10MB from the total bucket, shifts it into `ReservedBytes`, and returns `DIAMETER_SUCCESS (2001)` with a `Granted-Service-Unit` payload to the PCEF.
  * `UPDATE Request:` Once the chunk is consumed, PCEF reports usage. OCS purges 10MB from `ReservedBytes` to persistent memory and attempts to book the next 10MB chunk.
  * `TERMINATE Request:` If the session tears down and the user consumed only 4MB out of 10MB, the PCEF reports the remainder. OCS permanently deducts 4MB and **atomically refunds** the remaining 6MB back to `TotalBalanceBytes`.
* **[EN]** **Boundary Conditions:**
  * If `TotalBalanceBytes` < 10MB but > 0, OCS yields a "residual chunk" containing exactly the remaining byte balance.
  * If the balance is 0, OCS replies with a critical `DIAMETER_CREDIT_LIMIT_REACHED (4012)` status code, forcing the PCEF Core to immediately choke user bandwidth to 64 Kbps.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Все мутации балансов (`TotalBalanceBytes`, `ReservedBytes`) выполняются строго через **Lock-Free операции (`sync/atomic`)**, исключая использование мьютексов в ядре OCS [🧠].
2. Имитация списания и возврата остатков кванта при `TERMINATE` запросах работает со 100%-й точностью до 1 байта, проходя финансовый аудит в логах [🧠].
