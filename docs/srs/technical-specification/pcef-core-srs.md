# 🛡️ PCEF Core Engine — Technical Requirements Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Проектирование и реализация центрального высоконагруженного User Plane узла тарификации и шейпинга трафика. Модуль обязан на лету классифицировать L4-L7 пакеты, проверять финансовые квоты и применять QoS-политики на скоростях процессора без глобальных блокировок рантайма.
* **[EN]** Engineering and implementation of the central high-throughput User Plane packet valuation and traffic shaping hub. The module must classify L4-L7 frames on the fly, evaluate financial credit quotas, and enforce QoS constraints at bare-metal CPU speeds without triggering global runtime deadlocks.

### 📊 2. API Contracts & Bitmask Memory Layout / Контракты и Битовая маска в RAM
* **[RU]** Сетевое взаимодействие со шлюзом Access Gateway осуществляется через Full-Duplex gRPC Stream поверх протокола HTTP/2. Для обхода `if-else` условий используется упаковка состояний пакета в `uint64 Bitmask` по следующим маскам:
* **[EN]** Network interaction with the Access Gateway is driven by a binary Full-Duplex gRPC Stream over HTTP/2. To evade nested conditional statements, packet state evaluation is compressed into a single `uint64 Bitmask` layout:

```go
const (
	StateActive      uint64 = 1 << 0 // 0x01: Абонент успешно авторизован в сети
	TrafficStreaming uint64 = 1 << 1 // 0x02: DPI определил категорию YouTube/Streaming
	TrafficSocial    uint64 = 1 << 2 // 0x04: DPI определил категорию Мессенджеры/Social
	PackSmall        uint64 = 1 << 3 // 0x08: Диапазон фрейма до 1 Кбайт (Сигнализация/Пинг)
	PackHeavy        uint64 = 1 << 4 // 0x10: Диапазон фрейма тяжелого контента (Payload)
)
```

### ⚙️ 3. Algorithmic Logic & Hot Path Execution / Алгоритмы и физика обработки
1. **Эшелон Сегментации (Map Sharding):** поступивший фрейм по IP-адресу источника извлекается из **32-шардированного кэша сессий**. Каждый шард изолирован локальной `sync.RWMutex`, что снижает конкуренцию за замки памяти (*Mutex Contention*) до нуля при сотнях тысяч RPS [🧠].
2. **DPI Эшелон:** за константное время $O(1)$ по хэш-мапе сигнатур хоста/SNI вычисляется бит категории трафика [🧠].
3. **QoS Эшелон (Cache-Locality):** размер пакета `PayloadSizeBytes` прогоняется через плоский бинарный поиск `sort.Search` за время $O(\log N)$ по непрерывному массиву диапазонов. Монолитное расположение массива в памяти гарантирует его загрузку в L1/L2 кэш данных CPU, полностью исключая задержки на `Cache Miss` [🧠].
4. **Диспетчеризация (Table-Driven Dispatch):** результирующая маска вычисляется побитовым ИЛИ (`mask |= sizeBit`). Происходит O(1) вызов функции применения политики из реестра `map[uint64]PolicyAction` (пакет `internal/pkg/dispatch`). **Результат: полное избавление от 50k+ каскадов вложенных if-else условий, что намертво защищает CPU от сброса конвейера инструкций из-за ошибок предсказателя ветвлений (*Branch Misprediction*) [🧠]!**

### 🔄 4. Signalling & Protocols / Протоколы взаимодействия
* **Онлайн-тарификация (Интерфейс Gy):** синхронный gRPC-вызов в `ocs-rating` по протоколу **Diameter Gy (RFC 4006)** для резервирования/списания финансового кванта в 10 МБ.
* **Оффлайн-телеметрия (Интерфейс Gz):** асинхронный (в отдельной горутине, без блокировки горячего пути вычислений) пуш CDR-лога (Call Detail Record) по протоколу **Diameter Gz** в неблокирующий кольцевой буфер `message-bus` (Kafka) для пакетного сброса в ClickHouse [🧠].

### 🎖️ 5. Acceptance Criteria / Критерии приемки кода
* Обработка одного фрейма на горячем пути (включая Gy-запрос в OCS) занимает менее 1 миллисекунды ($p99$ latency < 1ms) [🧠].
* Любая незарегистрированная битовая комбинация (например, аномалия `0x15` — Social+Heavy) должна безопасно перехватываться движком с логированием ошибки через платформенный логер без краха рантайма Go [🧠].
