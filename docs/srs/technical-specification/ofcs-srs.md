# 📊 Offline Charging System (OFCS) — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Сбор, агрегация и пакетное сохранение асинхронных оффлайн-логов объемов трафика (CDR записей) в колоночную базу данных ClickHouse для долгосрочного бизнес-аудита СТО.
* **[EN]** Collection, aggregation, and batch mutation of asynchronous data plane volume logs (CDR entries) into a ClickHouse OLAP DBMS for deep analytical review by the CTO.

### 📊 2. Data Structures & Schema / Структуры данных логов
```go
type CallDetailRecord struct {
	RecordId     string // UUID лога
	SubscriberId string // IMSI абонента
	BytesDumped  int64  // Объем байт, пропущенный шлюзом
}
```

### ⚙️ 3. Non-blocking Buffering & Batch Insertion Mechanics / Алгоритмы пакетного сброса
* **[RU]** PCEF Core выстреливает логи в неблокирующий кольцевой буфер `message-bus` (Kafka) на базе Go-каналов емкостью 100 000 элементов через логику `select-default` [🧠]. Воркер `ofcs-collector` выгребает логи из очереди пачками [🧠]. Как только пачка собрана **строго по 30 штук ИЛИ прошел тайм-аут в 100 мс**, воркер производит пакетный `INSERT` в эмулятор ClickHouse, который сбрасывает блок монолитно в файл `data.bin` на диске вашего ПК, полностью устраняя деградацию дискового ввода-вывода [🧠].

### 🎖️ 4. Acceptance Criteria / Критерии приемки
1. Запись CDR в канал не блокирует сетевой поток обработки пакетов ядра (`Non-blocking I/O`) [🧠].
2. Логика сброса по таймауту работает стабильно через `time.Ticker` даже при снижении текущего трафика до нуля [🧠].
