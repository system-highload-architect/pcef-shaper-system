# 💸 Online Charging System (OCS) — Technical Requirements Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Эмуляция распределенного финтех-ядра реального времени по интерфейсу Gy Diameter. Главная b2b-задача — атомарное резервирование, списание и возврат неиспользованных остатков квот интернет-трафика (мегабайт) абонентов без использования блокирующих мьютексов операционной системы.
* **[EN]** Emulation of a highly concurrent real-time fintech core driving the Gy Diameter interface. Its primary business goal is the atomic reservation, commit, and refund of user data quotas (megabytes) with zero dependency on blocking operating system mutexes.

### 📊 2. Memory Data Structures / Структуры данных в RAM
```go
type GyCreditState struct {
	TotalBalanceBytes uint64 // Общий доступный счетчик байт абонента в RAM
	ReservedBytes     uint64 // Замороженный квант в рамках активной сессии
}
```

### ⚙️ 3. Algorithmic Logic & Quota Reservation Physics / Алгоритмы квантования
* **[RU]** **Паттерн Квантования (Credit Control State Machine):**
  * `INITIAL / UPDATE (Запрос кванта):` При проверке баланса, если `TotalBalanceBytes` $\ge 10$ МБ (`10*1024*1024` байт), OCS атомарно вычитает 10 МБ из общего счета, переносит их в `ReservedBytes` и возвращает в PCEF код `DIAMETER_SUCCESS (2001)`.
  * `UPDATE (Списание):` Когда квант выкачан, PCEF присылает отчет. OCS берет 10 МБ из `ReservedBytes` и списывает их окончательно, после чего бронирует следующие 10 МБ.
  * `TERMINATE (Возврат остатка):` Если сессия закрылась, а абонент выкачал только 4 МБ из 10, PCEF присылает отчет об использовании. OCS окончательно списывает 4 МБ, а неиспользованный остаток в 6 МБ **атомарно возвращает обратно** в `TotalBalanceBytes`, защищая деньги пользователя.
* **[RU]** **Пограничные условия (Boundary Conditions):**
  * Если `TotalBalanceBytes` меньше 10 МБ, но больше 0, OCS выдает «остаточный квант» (ровно столько байт, сколько осталось на балансе).
  * Если баланс равен 0, OCS возвращает финтех-код отсечки `DIAMETER_CREDIT_LIMIT_REACHED (4012)`, принуждая ядро PCEF запустить жесткий QoS-шейпинг до 64 Кбит/с.

### 🎖️ 4. Acceptance Criteria / Критерии приемки
1. Все финансовые мутации балансов (`TotalBalanceBytes`, `ReservedBytes`) выполняются строго через **Lock-Free операции (`sync/atomic`)**, исключая использование мьютексов в ядре OCS [🧠].
2. Имитация списания и возврата остатков кванта при `TERMINATE` запросах работает со 100%-й точностью до 1 байта, проходя финансовый аудит в логах [🧠].
