# 💸 Online Charging System (OCS) Specification with Aerospike Engine

### 🔍 Внутреннее устройство и прием данных / Mechanics & Data Ingestion
* **[RU]** OCS — это критический высоконагруженный финтех-сервер реального времени. Он управляет денежными и трафиковыми балансами. В качестве основного In-Memory хранилища применен **Aerospike Cluster**, работающий по гибридной архитектуре (Индексы в RAM, данные сессий напрямую на RAW NVMe блоках). Устройство базируется на паттерне **Резервирования квантов (Quota Allocation)** для снижения транзакционной нагрузки на диски.
* **[EN]** OCS is a critical, ultra-high-load real-time fintech engine managing financial and data balances. It utilizes a distributed **Aerospike Cluster** driven by Hybrid Memory Architecture (primary keys/indices reside in RAM, while session records point directly to RAW NVMe block devices). Its mechanics are built on the **Quota Allocation (Quantization)** pattern to mitigate transactional stress.

---

## ⏱️ Поток данных тарификации / Charging Data Sequence Flow

```mermaid
sequenceDiagram
    autonumber
    actor UE as 📱 User Equipment (Абонент)
    participant GW as 🌐 Access Gateway (Шлюз)
    participant PCEF as 🛡️ PCEF Core (Go Engine)
    participant OCS as 💸 OCS (Aerospike HMA)

    UE->>GW: Накатывается L4-L7 Трафик / Sends Data Packets
    GW->>PCEF: Перенаправление пакета в Shaper / Redirection to Shaper
    Note over PCEF: DPI определяет тип трафика<br/>DPI detects traffic signatures
    
    rect rgb(49, 151, 149)
        Note over PCEF,OCS: Цикл Diameter Gy Квантования / Diameter Gy Quantization Loop
        PCEF->>OCS: Credit-Control-Request (CCR: INITIAL / UPDATE)
        Note over OCS: Lock-Free проверка баланса на NVMe<br/>Lock-Free balance check on NVMe blocks
        OCS->>PCEF: Credit-Control-Answer (CCA: Granted-Service-Unit = 10MB)
    end

    alt Баланс успешный / Quota Granted
        PCEF->>GW: Разрешить пропуск на полной скорости / Allow full wire-speed
        GW->>UE: Доставка пакета завершена / Packet delivered successfully
    else Баланс равен 0 / Quota Exhausted (DIAMETER_CREDIT_LIMIT_REACHED)
        PCEF->>GW: Команда применения QoS: Включить Шейпинг 64 Kbps / Enforce Throttling to 64 Kbps
        GW->>UE: Скорость трафика жестко урезана / Traffic bandwidth strictly choked
    end
```

---

### ⚙️ Обработка и протоколы / Processing & Protocols
* **[RU]** Взаимодействие идет по интерфейсу **Gy (Diameter Credit-Control Application, RFC 4006)**. Движок OCS обрабатывает команды `CCR/CCA`. Использование Aerospike позволяет выполнять атомарные b2b-операции над балансами без блокировки всей таблицы абонентов благодаря механизму **Aerospike CDT (Complex Data Types)** и одношаговым операциям `Write/Increment` на уровне дискового контроллера.
* **[EN]** Interaction flows over the **Gy (Diameter Credit-Control Application, RFC 4006)** interface. It processes `CCR/CCA` command lifecycles. Utilizing Aerospike enables atomic b2b operations on credit balances without sweeping row locks via **Aerospike CDT (Complex Data Types)** and single-step atomic `Write/Increment` mutations at the controller layer.

### 🛠️ Выигрыш и Обоснование технологий / Technology Justification & Benefits
* **[RU]** **Технология: Aerospike Hybrid Memory Architecture (HMA).** Выигрыш: полное исключение оверхеда Сборщика Мусора (No Go/Java GC pauses inside storage). Достигается колоссальная экономия TCO инфраструктуры (до 80% расходов на RAM), так как терабайты данных квот лежат на дешевых NVMe SSD, но извлекаются со скоростью оперативной памяти благодаря прямому доступу ядра к блокам диска.
* **[EN]** **Technology: Aerospike Hybrid Memory Architecture (HMA).** Benefits: complete eradication of Storage-level Garbage Collection pauses. Yields massive infrastructure TCO compression (up to 80% RAM savings) because terabytes of quota logs reside on budget-friendly NVMe SSDs, while being retrieved at near-RAM speeds through raw kernel bypass block access.
