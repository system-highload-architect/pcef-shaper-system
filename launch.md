# 🏛️ 3GPP Policy & Charging Enforcement Function (PCEF) Shaper System

[RU] Данный модуль представляет собой высокопроизводительный, коммерческий User Plane движок PCEF (Функция применения политик и тарификации) с интегрированным DPI (Deep Packet Inspection) и QoS-шейпером трафика. Архитектура спроектирована по стандартам 3GPP PCC (Policy and Charging Control) для телеком- и финтех-экосистем.

[EN] This module implements a high-performance, production-ready User Plane PCEF (Policy & Charging Enforcement Function) engine featuring an integrated DPI (Deep Packet Inspection) classifier and QoS traffic shaper. Designed strictly according to 3GPP PCC (Policy and Charging Control) standards for telecom and fintech ecosystems.

---

## 🗺️ System Topology & Architecture / Архитектурная топология системы

```mermaid
graph T_D
    %% Стилизация элементов / Node Styling
    classDef control fill:#2b6cb0,stroke:#1a365d,stroke-width:2px,color:#fff;
    classDef user fill:#2f855a,stroke:#22543d,stroke-width:2px,color:#fff;
    classDef storage fill:#d69e2e,stroke:#744210,stroke-width:2px,color:#fff;
    classDef client fill:#4a5568,stroke:#2d3748,stroke-width:2px,color:#fff;

    %% Плоскость Управления / Control Plane (Left)
    subgraph Control_Plane [Control Plane / Плоскость управления]
        AF[Application Function / AF]:::control
        SPR[Subscription Profile Repository / SPR / UDR]:::storage
        PCRF[Policy & Charging Rules Function / PCRF]:::control
        OCS[Online Charging System / OCS]:::storage
        OFCS[Offline Charging System / OFCS]:::storage
    end

    %% Плоскость Пользователя / User Plane (Right)
    subgraph User_Plane [User Plane / Плоскость пользователя]
        UE[User Equipment / UE]:::client
        Gateway[Access Gateway / BNG / PGW / UPF]:::user
        
        subgraph PCEF_Core [PCEF Core Движок]
            DPI[🛡️ Internal DPI / Эшелон классификации трафика]:::user
            QoS[⚡ QoS Shaper / Шейпинг скорости]:::user
        end
        
        AAA[RADIUS / AAA Server]:::control
    end

    %% Взаимодействие в Control Plane / Control Plane Interactions
    AF -->|Rx / Diameter| PCRF
    SPR -->|Sp / Ud| PCRF
    
    %% Взаимодействие между Control и User Plane / Control-to-User Mapping
    PCRF -->|Gx / Diameter: Отправка PCC-правил| PCEF_Core
    
    %% Поток трафика пользователя / User Traffic Flow
    UE -->|L4-L7 Raw Traffic| Gateway
    Gateway -->|Data Packets| DPI
    
    %% Тарификация и Авторизация / Billing & Authorization
    PCEF_Core -->|Gy / Diameter: Онлайн-списания| OCS
    PCEF_Core -->|Gz / Diameter: Оффлайн-логи| OFCS
    PCEF_Core -->|Accounting Request| AAA
    Gateway -.->|Auth / Access| AAA

    %% Логические связи внутри PCEF / Internal logic pipeline
    DPI -->|Сигнатура трафика определена| QoS
```

---

## 📋 Technical Requirements Specification (SRS) / Техническое ТЗ проекта

[RU] Нашей b2b-задачей является реализация легковесного, отказоустойчивого эмулятора **User Plane PCEF** на чистом Go, абстрагированного от тяжелого Diameter-сериализатора, но на 100% повторяющего физику обработки L4-L7 фреймов под Highload-нагрузкой.

[EN] Our core objective is to build a lightweight, fault-tolerant **User Plane PCEF** emulator in pure Go. It abstracts away heavy Diameter serialization overhead while perfectly replicating L4-L7 packet processing physics under intense Highload stress.

### 1. Embedded DPI Classifier / Встроенный DPI-классификатор
* **[RU]** Сервер должен на лету парсить заголовки входящих сетевых пакетов. В рамках демо-кода классификация пакетов осуществляется по сигнатурам (Payload/Host Strings), разделяя трафик на три b2b-категории: `SOCIAL` (мессенджеры), `STREAMING` (тяжелое видео/YouTube) и `GAMING`.
* **[EN]** The engine must parse incoming network packet headers on the fly. In this demo-code scope, classification is driven by payload/host signatures, routing traffic into three distinct b2b categories: `SOCIAL` (messengers), `STREAMING` (heavy video/YouTube), and `GAMING`.

### 2. Credit Control Interface (Gy Sync) / Управление балансом в реальном времени
* **[RU]** Перед тем как пропустить пакет сквозь QoS-шлюз, PCEF обязан проверить баланс лицевого счета пользователя в OCS. Мы реализуем In-Memory аналог OCS на базе атомарных вычислений. Если у пользователя кончился пакет мегабайт или баланс равен 0, OCS возвращает код отсечки, и PCEF блокирует/срезает трафик.
* **[EN]** Prior to letting a packet through the QoS gateway, the PCEF must evaluate the subscriber's financial balance within the OCS. We implement an in-memory OCS subsystem utilizing atomic operations. If a subscriber exhausts their data quota or reaches a $0$ balance, the OCS returns a cutoff code, forcing the PCEF to throttle or drop the traffic.

### 3. Dynamic QoS Traffic Shaping / Динамический шейпинг скорости
* **[RU]** Применение политик ограничения скорости должно работать в реальном времени без глобальных блокировок рантайма Go. Мы применим усовершенствованный алгоритм **Leaky Bucket (Протекающее ведро)** для сглаживания всплесков трафика. Скорость пропускания байт (`Bandwidth Limit`) жестко регулируется PCC-правилами, полученными от эмулятора PCRF.
* **[EN]** Bandwidth throttling and traffic shaping must operate in real time without triggering global Go runtime deadlocks. We will deploy an optimized **Leaky Bucket** algorithm to smooth out network traffic spikes. The maximum byte throughput rate (`Bandwidth Limit`) is strictly enforced by PCC rules received from the PCRF emulator.

### 4. Highload Thread Isolation / Потокобезопасность ядра
* **[RU]** Обработка пакетов должна выполняться параллельными горутинами, утилизирующими все ядра CPU. Мапа сессий абонентов обязана исключать *Mutex Contention*. Мы применим паттерн **Map Sharding (Шардирование мап)** для снижения конкуренции за замки памяти под нагрузкой в сотни тысяч RPS.
* **[EN]** Packet processing must be driven by parallel goroutines utilizing all available CPU cores. The subscriber session map must eliminate *Mutex Contention*. We will deploy the **Map Sharding** pattern to reduce memory lock contention under loads exceeding hundreds of thousands of RPS.
