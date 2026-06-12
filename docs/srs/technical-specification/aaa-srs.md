# 🔒 RADIUS / AAA Server — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Эмуляция низкоуровневого UDP-сервера авторизации L4-сетевого стека для динамической привязки IP-адресов абонентов к их внутренним сессиям в RAM.
* **[EN]** Emulation of a low-level L4 network stack UDP authorization server for dynamic mapping of subscriber runtime IP addresses to internal memory session contexts.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type RadiusPacket struct {
	Code       byte     // 1 - Access-Request, 4 - Accounting-Request
	Identifier byte     // ID для сопоставления дубликатов пакетов
	Attributes map[byte][]byte // Карта AVP-атрибутов (IP, IMSI)
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** Сервер принимает сырой слайс байт, имитируя разбор сетевого кадра UDP. Поле `Attributes[8]` содержит `Framed-IP-Address` (4 байта IPv4), а `Attributes[1]` — строку `User-Name/IMSI` [🧠]. PCEF перехватывает этот пакет, вытаскивает метаданные и атомарно создает в оперативной памяти сессионную связку, проксируя пакет дальше в AAA [🧠].
* **[EN]** The server accepts a raw byte slice, mimicking a UDP network frame parse. Attribute key 8 maps to a `Framed-IP-Address` (4 bytes IPv4), while attribute 1 points to a `User-Name/IMSI` string string. PCEF intercepts this package, extracts the session metadata, and atomically creates a memory context before proxying the packet to the AAA.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Парсинг байт RADIUS-атрибутов выполняется без использования тяжелой рефлексии (`reflect`), строго через побайтовые смещения [🧠].
2. Время привязки IP к сессии абонента занимает менее 100 наносекунд [🧠].
