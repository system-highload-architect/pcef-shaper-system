# 🏛️ 3GPP Policy & Charging Enforcement Function (PCEF) Shaper System

[RU] Данный модуль представляет собой высокопроизводительный, коммерческий User Plane движок PCEF (функция применения политик и тарификации) с интегрированным DPI (Deep Packet Inspection) и QoS-шейпером трафика. Архитектура спроектирована по стандартам 3GPP PCC (Policy and Charging Control) для телеком- и финтех-экосистем.

[EN] This module implements a high-performance, production-ready User Plane PCEF (Policy & Charging Enforcement Function) engine featuring an integrated DPI (Deep Packet Inspection) classifier and QoS traffic shaper. Designed strictly according to 3GPP PCC (Policy and Charging Control) standards for telecom and fintech ecosystems.

### Навигация по архитектуре: [🚀 architecture](./docs/navigation.md)
### Навигация по техническому заданию: [🚀 technical specification](./docs//srs/srs-navigation.md)

---

## 🗺️ System Topology & Architecture / Архитектурная топология системы

```mermaid
graph TD
    %% Стилизация элементов / Node Styling
    classDef control fill:#2b6cb0,stroke:#1a365d,stroke-width:2px,color:#fff;
    classDef user fill:#2f855a,stroke:#22543d,stroke-width:2px,color:#fff;
    classDef storage fill:#d69e2e,stroke:#744210,stroke-width:2px,color:#fff;
    classDef client fill:#4a5568,stroke:#2d3748,stroke-width:2px,color:#fff;

    %% 1. Уровень Пользователей и Шлюзов / User & Gateway Layer
    UE[📱 User Equipment / UE]:::client
    Gateway[🌐 Access Gateway / BNG / PGW / UPF]:::user

    %% 2. Изолированный Движок PCEF Core / Isolated PCEF Core Engine
    subgraph PCEF_Core [PCEF Core Движок / Plane пользователя]
        DPI[🛡️ Internal DPI / Эшелон классификации]:::user
        QoS[⚡ QoS Shaper / Шейпинг скорости]:::user
    end

    %% 3. Уровень Управляющей Логики / Control Plane Layer
    PCRF[⚙️ PCRF / Функция управления политиками]:::control
    AF[🚀 Application Function / AF]:::control

    %% 4. Внешние Enterprise b2b Сервисы и СУБД / External b2b Services & Storage
    SPR[(🗄️ SPR / UDR / Профили абонентов)]:::storage
    OCS[(💸 OCS / Онлайн-тарификация Gy)]:::storage
    OFCS[(📊 OFCS / Оффлайн-логи Gz)]:::storage
    AAA[🔒 RADIUS / AAA Server]:::control

    %% СВЯЗИ И СИГНАЛИЗАЦИЯ / INTERFACES & SIGNALLING

    %% Сигнализация в Control Plane
    AF -->|Rx / Diameter| PCRF
    SPR -->|Sp / Ud| PCRF
    
    %% Интерфейс Gx (Управление)
    PCRF -->|Gx / Diameter: Отправка PCC-правил| DPI
    
    %% Поток трафика и RADIUS сессий от Шлюза
    UE -->|L4-L7 Raw Traffic| Gateway
    Gateway -->|1. RADIUS UDP Auth/Acct Requests| DPI
    Gateway -->|2. Data Packets / Зеркалирование| DPI

    %% Проксирование RADIUS и тарификация Diameter из PCEF Core
    DPI -->|RADIUS Proxy / Forward| AAA
    QoS -->|Gy / Diameter: Онлайн-списания| OCS
    QoS -->|Gz / Diameter: Оффлайн-логи| OFCS

    %% Внутренний конвейер PCEF
    DPI -->|3. Сигнатура трафика определена| QoS
```

---

## 📋 Technical Requirements Specification (SRS) / Общее техническое ТЗ проекта

[RU] Нашей b2b-задачей является реализация легковесного, отказоустойчивого эмулятора **User Plane PCEF** на чистом Go, абстрагированного от тяжелого Diameter-сериализатора, но на 100% повторяющего физику обработки L4-L7 фреймов под Highload-нагрузкой.

[EN] Our core objective is to build a lightweight, fault-tolerant **User Plane PCEF** emulator in pure Go. It abstracts away heavy Diameter serialization overhead while perfectly replicating L4-L7 packet processing physics under intense Highload stress.

### 1. Embedded DPI Classifier / Встроенный DPI-классификатор (Req. 1)
* **[RU]** Сервер должен на лету парсить заголовки входящих сетевых пакетов. В рамках демо-кода классификация пакетов осуществляется по сигнатурам (Payload/Host Strings), разделяя трафик на три b2b-категории: `SOCIAL` (мессенджеры), `STREAMING` (тяжелое видео/YouTube) и `GAMING`.
* **[EN]** The engine must parse incoming network packet headers on the fly. In this demo-code scope, classification is driven by payload/host signatures, routing traffic into three distinct b2b categories: `SOCIAL` (messengers), `STREAMING` (heavy video/YouTube), and `GAMING`.

### 2. Credit Control Interface (Gy Sync) / Управление балансом в реальном времени (Req. 2)
* **[RU]** Перед тем как пропустить пакет сквозь QoS-шлюз, PCEF обязан проверить баланс лицевого счета пользователя в OCS. Мы реализуем In-Memory аналог OCS на базе атомарных вычислений. Если у пользователя кончился пакет мегабайт или баланс равен 0, OCS возвращает код отсечки, и PCEF блокирует/срезает трафик.
* **[EN]** Prior to letting a packet through the QoS gateway, the PCEF must evaluate the subscriber's financial balance within the OCS. We implement an in-memory OCS subsystem utilizing atomic operations. If a subscriber exhausts their data quota or reaches a $0$ balance, the OCS returns a cutoff code, forcing the PCEF to throttle or drop the traffic.

### 3. Dynamic QoS Traffic Shaping / Динамический шейпинг скорости (Req. 3)
* **[RU]** Применение политик ограничения скорости должно работать в реальном времени без глобальных блокировок рантайма Go. Мы применим усовершенствованный алгоритм **Leaky Bucket (Протекающее ведро)** для сглаживания всплесков трафика. Скорость пропускания байт (`Bandwidth Limit`) жестко регулируется PCC-правилами, полученными от эмулятора PCRF.
* **[EN]** Bandwidth throttling and traffic shaping must operate in real time without triggering global Go runtime deadlocks. We will deploy an optimized **Leaky Bucket** algorithm to smooth out network traffic spikes. The maximum byte throughput rate (`Bandwidth Limit`) is strictly enforced by PCC rules received from the PCRF emulator.

### 4. Highload Thread Isolation / Потокобезопасность ядра (Req. 4)
* **[RU]** Обработка пакетов должна выполняться параллельными горутинами, утилизирующими все ядра CPU. Мапа сессий абонентов обязана исключать *Mutex Contention*. Мы применим паттерн **Map Sharding (Шардирование мап)** для снижения конкуренции за замки памяти под нагрузкой в сотни тысяч RPS.
* **[EN]** Packet processing must be driven by parallel goroutines utilizing all available CPU cores. The subscriber session map must eliminate *Mutex Contention*. We will deploy the **Map Sharding** pattern to reduce memory lock contention under loads exceeding hundreds of thousands of RPS.

### 5. Table-Driven Policy Dispatching / Табличная диспетчеризация политик (Req. 5)
* **[RU]** Для исключения деградации процессора на предсказании ветвлений (*Branch Misprediction*) при разрастании бизнес-логики до десятков тысяч условий, в ядро PCEF внедрен паттерн **Table-Driven Dispatch**. Каскады `if-else` и `switch` заменены на высокопроизводительный хэшированный реестр функций `map[string]PolicyAction`. Вычисление и применение PCC-правил происходит за константное время $O(1)$ через прямые вызовы кэшированных указателей функций в памяти RAM.
* **[EN]** To prevent CPU branch misprediction degradation as business logic scales to tens of thousands of conditions, the PCEF core deploys the **Table-Driven Dispatch** pattern. Nested `if-else` and `switch` cascades are replaced with a high-performance hashed function registry `map[string]PolicyAction`. Evaluation and enforcement of PCC rules take place within constant $O(1)$ time via direct execution of cached memory function pointers.

### 6. Hybrid Composite Key & Bitmask Matching / Композитная маршрутизация и битовые маски (Req. 6)
* **[RU]** Для обработки сложных составных условий (сочетание логических И/ИЛИ, равенств и интервалов вида `a < b && c != nil`) ядро PCEF полностью отказывается от вложенных Call Stack вызовов. Реализован двухэтапный конвейер: интервальные диапазоны вычисляются через плоский бинарный поиск $O(\log N)$, после чего стейт нормализуется и упаковывается в монолитный **Композитный ключ (Composite Dispatch Key)** или битовую маску (`uint64 Bitmask`). Итоговый выбор бизнес-логики сводится к единственной атомарной инструкции косвенного перехода в хэш-таблице функций, обеспечивая абсолютную чистоту архитектуры и нулевой уровень Mutex Contention.
* **[EN]** To process complex composite predicates (combinations of logical AND/OR, equalities, and range intervals like `a < b && c != nil`), the PCEF core completely eradicates nested Call Stack executions. A two-stage architecture pipeline is deployed: range intervals are evaluated via flat binary search $O(\log N)$, after which the active state is normalized and packed into a unified **Composite Dispatch Key** or a binary bitmask (`uint64 Bitmask`). The final business logic routing matches a single atomic indirect branch instruction within the function registry map, guaranteeing pure code architecture isolation and zero Mutex Contention.

---

## 🏛️ Общий технический разбор эшелонов архитектуры / Deep Architecture Deep Dive

### 1. Эшелон №1: точка входа трафика (Access Gateway, BNG, B_N_G, UPF)
* **Внутреннее устройство и физика процесса:** данный блок является физическим или виртуальным шлюзом терминации абонентских сессий (например, Broadband Network Gateway в фиксированных сетях или User Plane Function в сетях 5G). Он оперирует на уровнях L2/L3 сетевого стека. При подключении устройства пользователя (UE), шлюз инициирует RADIUS-сессию.
* **Протоколы и Взаимодействие:** 
  * `RADIUS UDP (Порты 1812/1813)`: направляет в PCEF Core пакеты `Access-Request` и `Accounting-Request` (Start/Interim/Stop). Внутри пакетов инкапсулированы атрибуты: `Framed-IP-Address` (выделенный IP), `Calling-Station-Id` (MSISDN/Идентификатор абонента) и `3GPP-User-Location-Info` (гео-локация).
  * `Raw IP L4-L7 Traffic`: зеркалирует или пропускает транзитом весь пользовательский трафик (Data Packets) напрямую в движок DPI через сетевые интерфейсы.
* **Выигрыш и Обоснование технологий:** на проде для пиковой пропускной способности интеграция шлюза с PCEF реализуется через технологию **DPDK (Data Plane Development Kit) или eBPF / XDP (Express Data Path)** в ядре Linux. Это позволяет Go-бэкенду забирать пакеты напрямую из кольцевого буфера сетевой карты (`Ring Buffer`), минуя тяжелый сетевой стек ядра Linux и исключая накладные расходы на переключение контекста CPU (*Context Switches*). Выигрыш: обработка пакетов со скоростью сетевой линии (*Line-Rate Processing*).

### 2. Эшелон №2: встроенный движок классификации трафика (Internal DPI Engine)
* **Внутреннее устройство и обработка данных:** получая поток сырых байт от шлюза, DPI (Deep Packet Inspection) не просто смотрит на IP/Порт (как классический L4-файрвол), а заглядывает в тело пакета (*Application Payload*). 
  * Он парсит первые 4-6 пакетов TCP-сессии (Паттерн *SSL/TLS Client Hello*), вытаскивая оттуда поле **SNI (Server Name Indication)**.
  * Если трафик зашифрован и SNI скрыт (TLS 1.3 ESNI), включается эвристический анализатор: проверяются поведенческие паттерны (размеры пакетов, джиттер, временные интервалы между фреймами).
* **Протоколы и Взаимодействие:**
  * `Gx Interface (Diameter RFC 6733 / 4006)`: асинхронно стучится к **PCRF**, используя уникальный идентификатор абонента (извлеченный из RADIUS). PCRF сверяется с репозиторием профилей (**SPR/UDR**) и возвращает в PCEF набор **PCC-правил (Policy and Charging Control Rules)**. В PCC-правилах жестко зашито: *«Для трафика с сигнатурой YouTube применить квоту Charging-Key=10, для мессенджеров Charging-Key=20»*.
* **Выигрыш и Обоснование технологий:** мапа сессий внутри DPI на Go реализуется через паттерн **Map Sharding (Шардирование)**. Вместо одной глобальной мапы под мьютексом, данные абонентов разбиваются на 256 независимых сегментов: `index = hash(IP) % 256`. Это полностью ликвидирует уязвимость *Mutex Contention* на Highload-нагрузках в сотни тысяч RPS. Выигрыш: Lock-Free чтение сессий абонентов за константное время $O(1)$ со стабильным Latency.

### 3. Эшелон №3: квантование и онлайн-тарификация (Online Charging System / OCS)
* **Внутреннее устройство и обработка данных:** движок OCS отвечает за финансовую стабильность платформы. Он работает в режиме реального времени по принципу **Квантования трафика (Quota Reservation)**. PCEF не списывает деньги за каждый байт (это убьет СУБД). Вместо этого PCEF запрашивает у OCS «квант» (резерв) данных — например, пакет размером в 10 Мегабайт. Пользователь качает трафик; как только 10 МБ исчерпаны, PCEF идет за следующим квантом.
* **Протоколы и Взаимодействие:**
  * `Gy Interface (Diameter Credit-Control Application)`: использует команды `CCR (Credit-Control-Request)` и `CCA (Credit-Control-Answer)`. Статусы сессии: `INITIAL` (запрос первого кванта при старте сессии), `UPDATE` (запрос следующего кванта по исчерпании), `TERMINATE` (возврат неиспользованного остатка кванта в OCS при закрытии сессии).
* **Выигрыш и Обоснование технологий:** хранилище балансов квантов переведено на **Aerospike Cluster с гибридной архитектурой памяти (Hybrid Memory Architecture)**. Мы полностью избавляемся от деградации памяти при раздувании кэша и исключаем задержки на репликацию. Операции проверки и декремента квот мегабайт выполняются как атомарные операции Aerospike CDT непосредственно на блочном уровне NVMe-дисков. Выигрыш: Строго предсказуемое Latency под жестким SLA (перцентиль p99 < 1 мс) при нагрузках >500 000 RPS, обеспечивающее стопроцентную финансовую надежность биллинга.

### 4. Эшелон №4: применение политик и динамический Шейпинг (QoS Shaper)
* **Внутреннее устройство и обработка данных:** если OCS подтвердил квоту, QoS-шейпер пропускает пакеты с максимальной скоростью согласно тарифу PCRF. Если OCS возвращает статус `DIAMETER_CREDIT_LIMIT_REACHED` (деньги кончились), QoS-шейпер мгновенно переключает стейт-машину абонента:
  * Либо полностью дропает (*Drop*) пакеты пользователя на уровне L3.
  * Либо включает **Traffic Shaping (Шейпинг)**, срезая скорость до гарантированного минимума (например, 64 Кбит/с), чтобы у абонента открывалась только страница пополнения баланса.
* **Протоколы и Взаимодействие:**
  * `Gz Interface (Diameter / File-based)`: асинхронно сливает CDR-файлы (Call Detail Records) и оффлайн-логи объемов скачанного трафика в **OFCS (Offline Charging System)** для последующего долгосрочного b2b-анализа и аудита в ClickHouse.
* **Выигрыш и Обоснование технологий:** алгоритм шейпинга пишется на базе паттерна **Leaky Bucket (Протекающее ведро)** со скользящим окном времени, без создания тяжелых фоновых горутин на каждого абонента (Lazy Refill). Обновление лимитов байт происходит реактивно, только в момент физического прилета сетевого пакета. Выигрыш: микроскопический Memory Footprint рантайма Go. Сервер удерживает миллионы активных QoS-сессий, расходуя считанные мегабайты памяти кучи, полностью защищая ноду от OOM (Out of Memory).

### 1. Tier №1: traffic entry point (Access Gateway, BNG, B_N_G, UPF)

* **Internal Architecture & Physical Process:** this block is a physical or virtual gateway that terminates subscriber sessions (e.g., Broadband Network Gateway in fixed networks or User Plane Function in 5G). it operates at L2/L3 of the network stack. when a User Equipment (UE) connects, the gateway initiates a RADIUS session.
* **Protocols & Interaction:**
  * `RADIUS UDP (Ports 1812/1813)`: sends `Access-Request` and `Accounting-Request` (Start/Interim/Stop) packets to the PCEF Core. inside the packets, attributes are encapsulated: `Framed-IP-Address` (allocated IP), `Calling-Station-Id` (MSISDN/subscriber identifier), and `3GPP-User-Location-Info` (geo-location).
  * `Raw IP L4-L7 Traffic`: mirrors or transparently forwards all user data packets directly to the DPI engine via network interfaces.
* **Technology Advantage & Justification:** in production, for peak throughput, the gateway integrates with PCEF using **DPDK (Data Plane Development Kit) or eBPF / XDP (Express Data Path)** in the Linux kernel. this allows the Go backend to pull packets directly from the NIC’s ring buffer, bypassing the heavy Linux kernel network stack and eliminating CPU context‑switch overhead. **Benefit:** line‑rate packet processing.

### 2. Tier №2: built‑in traffic classification engine (Internal DPI Engine)

* **Internal Architecture & Data Processing:** receiving a raw byte stream from the gateway, DPI (Deep Packet Inspection) does not merely look at IP/port (unlike a classic L4 firewall) but inspects the application payload.
  * it parses the first 4–6 packets of a TCP session (e.g., SSL/TLS Client Hello) to extract the **SNI (Server Name Indication)** field.
  * if traffic is encrypted and SNI is hidden (TLS 1.3 ESNI), a heuristic analyzer takes over: behavioural patterns are checked (packet sizes, jitter, time intervals between frames).
* **Protocols & Interaction:**
  * `Gx Interface (Diameter RFC 6733 / 4006)`: asynchronously contacts the PCRF, using the unique subscriber identifier (extracted from RADIUS). the PCRF consults the subscriber profile repository (**SPR/UDR**) and returns a set of **PCC rules (Policy and Charging Control Rules)** to the PCEF. each PCC rule rigidly defines: *“for traffic with YouTube signature apply quota Charging‑Key=10; for messengers Charging‑Key=20”*.
* **Technology Advantage & Justification:** the session map inside the DPI is implemented in Go using **Map Sharding**. instead of a single global map under a mutex, subscriber data is split into 256 independent segments: `index = hash(IP) % 256`. this completely eliminates *mutex contention* under high load (hundreds of thousands of RPS). **Benefit:** lock‑free, O(1) constant‑time reading of subscriber sessions with stable latency.

### 3. Tier №3: quota management & online charging (Online Charging System / OCS)

* **Internal Architecture & Data Processing:** the OCS engine ensures the financial stability of the platform. it operates in real time using the **Quota Reservation** principle. the PCEF does not debit money per byte (that would kill the database). instead, it requests a “quota” (reservation) of data from the OCS – for example, a 10‑megabyte bucket. the user consumes traffic; as soon as the 10 MB is exhausted, the PCEF requests the next quota.
* **Protocols & Interaction:**
  * `Gy Interface (Diameter Credit‑Control Application)`: uses `CCR (Credit‑Control‑Request)` and `CCA (Credit‑Control‑Answer)` commands. session states: `INITIAL` (request for first quota at session start), `UPDATE` (request for next quota when exhausted), `TERMINATE` (return unused quota balance to OCS when session ends).
* **Technology Advantage & Justification:** the quota balance storage is moved to an **Aerospike Cluster with Hybrid Memory Architecture**. this eliminates memory degradation due to cache bloat and removes replication delays. checks and decrements of megabyte quotas are performed as Aerospike CDT atomic operations directly at the NVMe block level. **Benefit:** strictly predictable latency under hard SLA (p99 < 1 ms) at loads >500,000 RPS, ensuring 100% financial reliability of billing.

### 4. Tier №4: policy enforcement & dynamic shaping (QoS Shaper)

* **Internal Architecture & Data Processing:** if the OCS confirms the quota, the QoS shaper forwards packets at the maximum rate allowed by the PCRF tariff. if the OCS returns a `DIAMETER_CREDIT_LIMIT_REACHED` status (funds exhausted), the QoS shaper instantly changes the subscriber’s state machine:
  * either completely drops the user’s packets at L3,
  * or activates **Traffic Shaping**, reducing the rate to a guaranteed minimum (e.g., 64 Kbps) so that only a balance top‑up page is reachable.
* **Protocols & Interaction:**
  * `Gz Interface (Diameter / File‑based)`: asynchronously streams CDR (Call Detail Record) files and offline logs of downloaded traffic to the **OFCS (Offline Charging System)** for long‑term B2B analysis and auditing in ClickHouse.
* **Technology Advantage & Justification:** the shaping algorithm is implemented using the **Leaky Bucket** pattern with a sliding time window, without creating heavy background goroutines per subscriber (lazy refill). byte‑limit updates happen reactively only when a network packet actually arrives. **Benefit:** microscopic memory footprint for the Go runtime. a server can hold millions of active QoS sessions using only a few megabytes of heap memory, fully protecting the node from OOM (Out‑of‑Memory).