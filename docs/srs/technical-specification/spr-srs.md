# 🗄️ Subscription Profile Repository (SPR / UDR) — Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Имитация высоконагруженной NoSQL СУБД для хранения сотен тысяч b2b-паспортов контрактов абонентов с микросекундной скоростью извлечения данных в RAM кучи.
* **[EN]** Emulation of a highly-scalable NoSQL DBMS storing hundreds of thousands of subscriber enterprise contract passports with microsecond-level RAM data retrieval.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type SubscriberProfile struct {
	IMSI         string   `json:"imsi"`          // Ключ шардирования (MSISDN ID)
	TariffClass  string   `json:"tariff_class"`  // "VIP", "BASE", "IOT"
	IsSuspended  bool     `json:"is_suspended"`  // Флаг принудительной финансовой блокировки
	GrantedBytes int64    `json:"granted_bytes"` // Стартовый объем интернет-пакета в байтах
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** Для исключения гонок данных (*Data Race*) и *Mutex Contention*, база данных эмулируется через паттерн **Map Sharding** [🧠]. Весь пул абонентов бьется на 32 сегмента, каждый под своей изолированной `sync.RWMutex` [🧠]. Поиск сегмента: `shardIndex := murmur3.Sum32(imsi) % 32` [🧠]. Чтение профиля выполняется через конкурентный `RLock` [🧠].
* **[EN]** To mitigate data races and lock contention, the repository is emulated via the **Map Sharding** pattern. The entire subscriber pool is split into 32 autonomous shards, each guarded by its isolated `sync.RWMutex`. Shard routing is computed as `shardIndex := hash(imsi) % 32`. Profile read execution utilizes parallel `RLock` threads.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Время поиска абонента среди 100 000 записей в RAM не превышает 3 микросекунды [🧠].
2. Тест конкурентности `go test -race` подтверждает полное отсутствие гонок данных под нагрузкой [🧠].
