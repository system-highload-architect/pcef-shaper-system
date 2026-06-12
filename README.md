# 🏛️ 3GPP Policy & Charging Enforcement Function (PCEF) Shaper System

[RU] Данный модуль представляет собой высокопроизводительный, коммерческий User Plane движок PCEF (Функция применения политик и тарификации) с интегрированным DPI (Deep Packet Inspection) и QoS-шейпером трафика. Архитектура спроектирована по стандартам 3GPP PCC (Policy and Charging Control) для телеком- и финтех-экосистем.

[EN] This module implements a high-performance, production-ready User Plane PCEF (Policy & Charging Enforcement Function) engine featuring an integrated DPI (Deep Packet Inspection) classifier and QoS traffic shaper. Designed strictly according to 3GPP PCC (Policy and Charging Control) standards for telecom and fintech ecosystems.

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

## 📋 Technical Requirements Specification (SRS) / Техническое ТЗ проекта

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

---

## 🏛️ Глубокий технический разбор эшелонов архитектуры / Deep Architecture Deep Dive

### 1. Эшелон №1: точка входа трафика (Access Gateway, BNG, B_N_G, UPF)
* **Внутреннее устройство и физика процесса:** Данный блок является физическим или виртуальным шлюзом терминации абонентских сессий (например, Broadband Network Gateway в фиксированных сетях или User Plane Function в сетях 5G). Он оперирует на уровнях L2/L3 сетевого стека. При подключении устройства пользователя (UE), шлюз инициирует RADIUS-сессию.
* **Протоколы и Взаимодействие:** 
  * `RADIUS UDP (Порты 1812/1813)`: направляет в PCEF Core пакеты `Access-Request` и `Accounting-Request` (Start/Interim/Stop). Внутри пакетов инкапсулированы атрибуты: `Framed-IP-Address` (выделенный IP), `Calling-Station-Id` (MSISDN/Идентификатор абонента) и `3GPP-User-Location-Info` (гео-локация).
  * `Raw IP L4-L7 Traffic`: зеркалирует или пропускает транзитом весь пользовательский трафик (Data Packets) напрямую в движок DPI через сетевые интерфейсы.
* **Выигрыш и Обоснование технологий:** На проде для пиковой пропускной способности интеграция шлюза с PCEF реализуется через технологию **DPDK (Data Plane Development Kit) или eBPF / XDP (Express Data Path)** в ядре Linux. Это позволяет Go-бэкенду забирать пакеты напрямую из кольцевого буфера сетевой карты (`Ring Buffer`), минуя тяжелый сетевой стек ядра Linux и исключая накладные расходы на переключение контекста CPU (*Context Switches*). Выигрыш: обработка пакетов со скоростью сетевой линии (*Line-Rate Processing*).

### 2. Эшелон №2: встроенный движок классификации трафика (Internal DPI Engine)
* **Внутреннее устройство и обработка данных:** Получая поток сырых байт от шлюза, DPI (Deep Packet Inspection) не просто смотрит на IP/Порт (как классический L4-файрвол), а заглядывает в тело пакета (*Application Payload*). 
  * Он парсит первые 4-6 пакетов TCP-сессии (Паттерн *SSL/TLS Client Hello*), вытаскивая оттуда поле **SNI (Server Name Indication)**.
  * Если трафик зашифрован и SNI скрыт (TLS 1.3 ESNI), включается эвристический анализатор: проверяются поведенческие паттерны (размеры пакетов, джиттер, временные интервалы между фреймами).
* **Протоколы и Взаимодействие:**
  * `Gx Interface (Diameter RFC 6733 / 4006)`: Асинхронно стучится к **PCRF**, используя уникальный идентификатор абонента (извлеченный из RADIUS). PCRF сверяется с репозиторием профилей (**SPR/UDR**) и возвращает в PCEF набор **PCC-правил (Policy and Charging Control Rules)**. В PCC-правилах жестко зашито: *«Для трафика с сигнатурой YouTube применить квоту Charging-Key=10, для мессенджеров Charging-Key=20»*.
* **Выигрыш и Обоснование технологий:** Мапа сессий внутри DPI на Go реализуется через паттерн **Map Sharding (Шардирование)**. Вместо одной глобальной мапы под мьютексом, данные абонентов разбиваются на 256 независимых сегментов: `index = hash(IP) % 256`. Это полностью ликвидирует уязвимость *Mutex Contention* на Highload-нагрузках в сотни тысяч RPS. Выигрыш: Lock-Free чтение сессий абонентов за константное время $O(1)$ со стабильным Latency.

### 3. Эшелон №3: квантование и Онлайн-тарификация (Online Charging System / OCS)
* **Внутреннее устройство и обработка данных:** Движок OCS отвечает за финансовую стабильность платформы. Он работает в режиме реального времени по принципу **Квантования трафика (Quota Reservation)**. PCEF не списывает деньги за каждый байт (это убьет СУБД). Вместо этого PCEF запрашивает у OCS «квант» (резерв) данных — например, пакет размером в 10 Мегабайт. Пользователь качает трафик; как только 10 МБ исчерпаны, PCEF идет за следующим квантом.
* **Протоколы и Взаимодействие:**
  * `Gy Interface (Diameter Credit-Control Application)`: использует команды `CCR (Credit-Control-Request)` и `CCA (Credit-Control-Answer)`. Статусы сессии: `INITIAL` (запрос первого кванта при старте сессии), `UPDATE` (запрос следующего кванта по исчерпании), `TERMINATE` (возврат неиспользованного остатка кванта в OCS при закрытии сессии).
* **Выигрыш и Обоснование технологий:** Хранилище балансов квантов переведено на **Aerospike Cluster с гибридной архитектурой памяти (Hybrid Memory Architecture)**. Мы полностью избавляемся от деградации памяти при раздувании кэша и исключаем задержки на репликацию. Операции проверки и декремента квот мегабайт выполняются как атомарные операции Aerospike CDT непосредственно на блочном уровне NVMe-дисков. Выигрыш: Строго предсказуемое Latency под жестким SLA (перцентиль p99 < 1 мс) при нагрузках >500 000 RPS, обеспечивающее стопроцентную финансовую надежность биллинга.

### 4. Эшелон №4: применение политик и Динамический Шейпинг (QoS Shaper)
* **Внутреннее устройство и обработка данных:** Если OCS подтвердил квоту, QoS-шейпер пропускает пакеты с максимальной скоростью согласно тарифу PCRF. Если OCS возвращает статус `DIAMETER_CREDIT_LIMIT_REACHED` (деньги кончились), QoS-шейпер мгновенно переключает стейт-машину абонента:
  * Либо полностью дропает (*Drop*) пакеты пользователя на уровне L3.
  * Либо включает **Traffic Shaping (Шейпинг)**, срезая скорость до гарантированного минимума (например, 64 Кбит/с), чтобы у абонента открывалась только страница пополнения баланса.
* **Протоколы и Взаимодействие:**
  * `Gz Interface (Diameter / File-based)`: асинхронно сливает CDR-файлы (Call Detail Records) и оффлайн-логи объемов скачанного трафика в **OFCS (Offline Charging System)** для последующего долгосрочного b2b-анализа и аудита в ClickHouse.
* **Выигрыш и Обоснование технологий:** Алгоритм шейпинга пишется на базе паттерна **Leaky Bucket (Протекающее ведро)** со скользящим окном времени, без создания тяжелых фоновых горутин на каждого абонента (Lazy Refill). Обновление лимитов байт происходит реактивно, только в момент физического прилета сетевого пакета. Выигрыш: микроскопический Memory Footprint рантайма Go. Сервер удерживает миллионы активных QoS-сессий, расходуя считанные мегабайты памяти кучи, полностью защищая ноду от OOM (Out of Memory).
