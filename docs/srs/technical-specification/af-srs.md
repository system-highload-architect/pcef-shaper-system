# 🚀 Application Function (AF) — Technical Requirements Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Имитация поведения контент-провайдеров (IPTV/VOD) для динамического изменения сетевых политик (QoS) абонента на уровне ядра под конкретные медиа-сессии.
* **[EN]** Simulation of content provider platforms (IPTV/VOD) to dynamically mutate a subscriber's network quality policies (QoS) for specific high-bandwidth media sessions.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type MediaSessionRequest struct {
	SubscriberID string    `json:"subscriber_id"` // Идентификатор абонента (MSISDN)
	MediaIP      string    `json:"media_ip"`      // IP-адрес видео-сервера CDN
	RequiredQoS  string    `json:"required_qos"`  // Класс обслуживания: "VIDEO_4K", "VOIP"
	Duration     time.Duration `json:"duration"`  // Длительность бронирования полосы
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** При вызове эмулятора AF, система обязана сформировать Diameter Rx `AA-Request (AAR)` пакет. Если `RequiredQoS` содержит неизвестную строку, система возвращает ошибку `DIAMETER_INVALID_AVP_VALUE (5004)` [🧠]. Асинхронно запускается таймер: по истечении `Duration` AF обязан послать команду `STR (Session-Termination-Request)`, сбрасывающую приоритет [🧠].
* **[EN]** Upon AF execution, the system must assemble a Diameter Rx `AA-Request (AAR)` packet. If `RequiredQoS` contains an invalid payload string, it falls back to a `DIAMETER_INVALID_AVP_VALUE (5004)` error. A runtime timer is spawned asynchronously: upon `Duration` expiration, AF must transmit an `STR` command to revoke privileges.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Время упаковки gRPC-запроса в Diameter Rx структуру в памяти кучи не превышает 500 наносекунд [🧠].
2. Таймер отмены сессии работает без утечек горутин (`Goroutine Leak`) через `time.AfterFunc` [🧠].
