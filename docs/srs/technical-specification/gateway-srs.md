# 🌐 Access Gateway (BNG / PGW / UPF) — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Имитация силовой L3/L4 точки прохода сетевого трафика. Проброс байт сквозь встроенный eBPF/XDP Kernel-Bypass слой в движок PCEF для проверки квот и политик.
* **[EN]** Emulation of a high-throughput L3/L4 data execution plane junction. Streaming byte payloads through an embedded eBPF/XDP Kernel-Bypass driver directly into the PCEF shaper core.

### 📊 2. Data Structures & Contract / Структуры данных и контракт на Go
```go
type NetworkPacketFrame struct {
	SourceIP        string // От кого летит трафик (IP абонента)
	DestinationHost string // Куда летит (например, "youtube.com")
	PayloadSizeByte int    // Физический размер пакета в байтах
}
```

### ⚙️ 3. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** Шлюз работает как сквозной конвейер. Он принимает `NetworkPacketFrame`, на лету запрашивает вердикт у PCEF Core. Если PCEF возвращает скорость = 0, шлюз имитирует системную команду `XDP_DROP`, уничтожая пакет [🧠]. Если скорость ограничена, шлюз отправляет пакет в микро-буфер алгоритма `Leaky Bucket` для сглаживания джиттера [🧠].
* **[EN]** The gateway functions as a transparent pipeline. It accepts a `NetworkPacketFrame` and evaluates a runtime verdict from the PCEF Core on the fly. If PCEF returns a 0 bandwidth limit, the gateway triggers an `XDP_DROP` system command simulation, discarding the packet. If throttled, it passes it to a `Leaky Bucket` micro-buffer.

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Проброс пакета через шлюз выполняется за линейное время $O(1)$ [🧠].
2. Имитация сброса `XDP_DROP` не расходует такты CPU на аллокацию памяти под тело пакета [🧠].
