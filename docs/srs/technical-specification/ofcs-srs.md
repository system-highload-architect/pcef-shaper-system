# 📊 Offline Charging System (OFCS) — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Высокопроизводительный асинхронный сборщик оффлайн-логов и CDR записей для последующей заливки в ClickHouse OLAP СУБД с минимальным Memory Footprint рантайма.
* **[EN]** High-performance asynchronous offline telemetry and CDR log aggregator tailored for ClickHouse OLAP DBMS batch streaming with minimal runtime memory footprint.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type CallDetailRecord struct {
	RecordID    string    `json:"record_id"`   // UUID лога
	SubscriberID string   `json:"sub_id"`      // Кто качал трафик
	BytesDumped int64     `json:"bytes_dump"`  // Сколько байт прогнал шлюз
	Timestamp   time.Time `json:"timestamp"`   // Время фиксации
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** PCEF Core шлет CDR записи в неблокирующий кольцевой буфер (эмулятор Apache Kafka) на базе Go-каналов (`chan CallDetailRecord`) емкостью 100 000 элементов [🧠]. Фоновые воркеры выгребают логи пачками (**Batching**) ровно по 30 штук [🧠]. Как только пачка собрана ИЛИ прошел тайм-аут в 100мс, воркер производит имитацию пакетного сброса на диск СУБД [🧠].
* **[EN]** The PCEF Core pushes CDR data into a non-blocking circular buffer (Apache Kafka emulator) driven by Go channels (`chan CallDetailRecord`) with a capacity of 100,000 items. Concurrent workers batch read logs in chunks of exactly 30 items. Once a batch is assembled OR a 100ms flush timeout is reached, the worker triggers a simulated database block write.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Запись CDR в канал не блокирует сетевой поток обработки пакетов (`Non-blocking I/O`) [🧠].
2. Логика сброса по таймауту работает стабильно через `time.Ticker` даже при нулевом текущем трафике [🧠].
