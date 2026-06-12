# 📋 PCEF Technical Specification (SRS) Index / Индекс технических требований ТЗ

[RU] Этот документ является единым навигационным пультом управления техническими требованиями (SRS) для каждого компонента системы PCEF. Все спецификации строго синхронизированы с архитектурными диаграммами последовательностей.

[EN] This document serves as a unified Technical Requirements Specification (SRS) index for each component of the PCEF system. All specifications are strictly synchronized with the architectural sequence flow charts.

---

## 🏗️ Control Plane Requirements / Требования к плоскости управления

*   ### [🚀 AF Requirements (Требования к Application Function)](./technical-specification/af-srs.md)
    *   [RU] Спецификация генерации медиа-запросов, форматы Diameter Rx пакетов, имитация SLA.
    *   [EN] Media request generation spec, Diameter Rx packet formats, SLA simulation rules.
*   ### [⚙️ PCRF Requirements (Требования к Policy & Charging Engine)](./technical-specification/pcrf-srs.md)
    *   [RU] Алгоритмы Rete-компиляции, структуры PCC-правил, сопоставление тарифов Sp/Gx.
    *   [EN] Rete compilation algorithms, PCC rules data structures, Sp/Gx tariff mapping matrix.
*   ### [🗄️ SPR/UDR Requirements (Требования к Базе профилей)](./technical-specification/spr-srs.md)
    *   [RU] Схемы данных профилей абонентов, индексы Key-Value, симуляция LSM-дерева в RAM.
    *   [EN] Subscriber profile schemas, Key-Value lookup indices, in-memory LSM-tree simulation.
*   ### [💸 OCS Requirements (Требования к Онлайн-биллингу Aerospike)](./technical-specification/ocs-srs.md)
    *   [RU] Алгоритмы квантования Gy, CCR/CCA стейт-машины, атомарные финансовые транзакции.
    *   [EN] Gy quantization algorithms, CCR/CCA state machines, atomic transaction mutations.
*   ### [📊 OFCS Requirements (Требования к Оффлайн-логам ClickHouse)](./technical-specification/ofcs-srs.md)
    *   [RU] Пакетная агрегация CDR (Batching), асинхронные очереди, структуры буферов Kafka.
    *   [EN] CDR batch aggregation, asynchronous queue structures, Kafka buffer simulations.

---

## 🟢 User Plane Requirements / Требования к плоскости пользователя

*   ### [🔒 RADIUS / AAA Requirements (Требования к Серверу авторизации)](./technical-specification/aaa-srs.md)
    *   [RU] Парсинг UDP RADIUS пакетов, извлечение IP сессий, управление IPAM пулами.
    *   [EN] UDP RADIUS packet parsing, IP session tracking, IPAM pool allocation.
*   ### [🌐 Access Gateway Requirements (Требования к Сетевому Шлюзу)](./technical-specification/gateway-srs.md)
    *   [RU] Имитация eBPF/XDP перехвата, Kernel-Bypass буферы байт, транзит пакетов.
    *   [EN] eBPF/XDP intercept emulation, Kernel-Bypass byte buffers, packet transit pipelines.
*   ### [📱 UE Requirements (Требования к Эмулятору Абонентов)](./technical-specification/ue-srs.md)
    *   [RU] Многопоточная генерация трафика горутинами, профили джиттера, стресс-бенчмарки.
    *   [EN] Highly concurrent traffic generation, jitter network profiles, stress-test benchmarks.
