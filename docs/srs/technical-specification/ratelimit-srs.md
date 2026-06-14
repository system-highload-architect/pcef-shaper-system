# ⚡ Lock-Free Rate Limiter Chassis — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Реализация пуленепробиваемого, отказоустойчивого эшелона защиты L7-уровня (WAF/Файрвол) для предотвращения DDoS-атак и флуда запросами. Компонент обязан ограничивать частоту gRPC-пакетов на входе в исполнительное ядро PCEF со скоростью аппаратных инструкций CPU без использования блокирующих мьютексов.
* **[EN]** Engineering and deployment of a bulletproof L7 rate-limiting shield (WAF/Firewall) to actively mitigate DDoS floods. The component must restrict gRPC request frequency at hardware CPU speeds, completely bypassing operating system mutex overhead.

### 📊 2. Memory Data Structures & Scaled Types / Структуры данных в RAM
* **[RU]** Для исключения накладных расходов на вычисления с плавающей точкой (`float64`) на горячем пути, состояние маркерной корзины (*Token Bucket*) масштабируется в 64-битные целые числа (`int64`):
* **[EN]** To evade float64 execution boundaries on the hot execution path, the Token Bucket state is preserved via scaled 64-bit integer primitives:

```go
type AtomicClientBucket struct {
	LastRefillNano int64 // Таймстамп последнего пополнения в наносекундах UnixNano
	TokensScaled   int64 // Текущие токены, умноженные на 1 000 000 для точности дроби в int64
}
```

### ⚙️ 3. Hardware CAS Loops & Spin-Lock Mitigation / Логика Lock-Free конвейера
1. **Атомное пополнение (Lazy Refill):** При каждом запросе время простоя вычисляется в наносекундах (`now - oldRefill`). Математика прихода токенов рассчитывается атомарно и масштабируется [🧠].
2. **CAS-Синхронизация:** Мутация баланса токенов и списание одного маркера происходят в бесконечном цикле `for` через аппаратную инструкцию процессора **Compare-And-Swap (`atomic.CompareAndSwapInt64`)** [🧠]. Мьютексы полностью стерты из RAM, горутины никогда не засыпают в ожидании освобождения замка [🧠].
3. ** runtime.Gosched() Backoff:** Для предотвращения выжигания ядер CPU холостым вращением потоков (*Spin-Lock Starvation*) при агрессивном DDoS-штурме, сорвавшийся в CAS-дуэли поток мягко уступает текущий квант времени ОС соседней горутине через системный вызов **`runtime.Gosched()`** [🧠].
4. **WAF Вердикт:** Если токенов в бакете недостаточно, лимитер возвращает ошибку, и gRPC-транспорт ядра PCEF Core мгновенно отправляет шлюзу команду **`XDP_DROP` со скоростью 0 бит/с**, полностью изолируя тяжелую бизнес-логику от флуда [🧠].

### 🎖️ 4. Acceptance Criteria / Критерии приемки кода
1. Пропускная способность лимитера составляет >1 000 000 RPS на одно ядро CPU без деградации и блокировок потоков ОС [🧠].
2. Слой телеметрии обязан атомарно инкрементировать метрику `pcef_blocked_packets_total` в OpenTelemetry API при каждом отражении атаки [🧠].
