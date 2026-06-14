# 🗄️ Subscription Profile Repository (SPR / UDR) — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Проектирование и реализация NoSQL хранилища контрактов абонентов со сверхнизкой задержкой извлечения данных в RAM кучи по запросам от Control Plane плоскости (PCRF движка).
* **[EN]** Engineering and implementation of a low-latency NoSQL repository for subscriber contracts, optimizing memory footprint and data retrieval inside the Go heap upon Control Plane (PCRF) triggers.

### 📊 2. Go Data Structs & Contracts / Структуры данных на Go
```go
type SubscriberProfile struct {
	IMSI         string `json:"imsi"`          // Уникальный ID сим-карты (Ключ шардирования)
	TariffClass  string `json:"tariff_class"`  // Группа тарифа: "VIP", "BASE", "IOT"
	IsSuspended  bool   `json:"is_suspended"`  // Флаг принудительной финансовой блокировки
	GrantedBytes int64  `json:"granted_bytes"` // Стартовый объем интернет-пакета в байтах
}
```

### ⚙️ 3. Map Sharding Logic & Boundary Conditions / Шардирование и Алгоритмы
* **[RU]** Для полной ликвидации *Mutex Contention* под параллельным натиском горутин PCRF-движка, хранилище разбито на 32 независимых сегмента памяти (`ProfileShard`) [🧠]. Маршрутизация ключа выполняется через вычисление хэша FNV-1a: `shardIndex := fnv.New32a(imsi) % 32` [🧠]. Каждый шард защищен своей локальной `sync.RWMutex`, позволяя миллионам горутин параллельно читать данные без блокировок через `RLock` [🧠].

### 🎖️ 4. Acceptance Criteria / Критерии приемки
1. Время поиска абонента среди 100 000 записей в RAM не превышает 3 микросекунды [🧠].
2. Тест конкурентности `go test -race` подтверждает полное отсутствие гонок данных под нагрузкой [🧠].
