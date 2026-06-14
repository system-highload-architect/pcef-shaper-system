# 📋 PCEF Technical Specification (SRS) Index / Индекс технических требований ТЗ

[RU] Этот документ является единым навигационным пультом управления техническими требованиями (SRS) для каждого компонента системы PCEF. Все спецификации строго синхронизированы с архитектурными диаграммами последовательностей.

[EN] This document serves as a unified Technical Requirements Specification (SRS) index for each component of the PCEF system. All specifications are strictly synchronized with the architectural sequence flow charts.

---

## 🏗️ Control Plane Requirements / Требования к плоскости управления

*   ### [🚀 AF Requirements (Требования к Application Function)](./technical-specification/af-srs.md)
    *   [RU] Спецификация генерации медиа-запросов, форматы Diameter Rx пакетов, имитация SLA.
*   ### [⚙️ PCRF Requirements (Требования к Policy & Charging Engine)](./technical-specification/pcrf-srs.md)
    *   [RU] Алгоритмы Rete-компиляции, структуры PCC-правил, сопоставление тарифов Sp/Gx.
*   ### [🗄️ SPR/UDR Requirements (Требования к Базе профилей)](./technical-specification/spr-srs.md)
    *   [RU] Схемы данных профилей абонентов, индексы Key-Value, симуляция LSM-дерева в RAM.
*   ### [💸 OCS Requirements (Требования к Онлайн-биллингу Aerospike)](./technical-specification/ocs-srs.md)
    *   [RU] Алгоритмы квантования Gy, CCR/CCA стейт-машины, атомарные финансовые транзакции.
*   ### [📊 OFCS Requirements (Требования к Оффлайн-логам ClickHouse)](./technical-specification/ofcs-srs.md)
    *   [RU] Пакетная агрегация CDR (Batching), асинхронные очереди, структуры буферов Kafka.

---

## 🟢 User Plane & Platform Core Requirements / Требования к ядру и пользователю

*   ### [🔒 RADIUS / AAA Requirements (Требования к Серверу авторизации)](./technical-specification/aaa-srs.md)
    *   [RU] Парсинг UDP RADIUS пакетов, извлечение IP сессий, управление IPAM пулами.
*   ### [🌐 Access Gateway Requirements (Требования к Сетевому Шлюзу)](./technical-specification/gateway-srs.md)
    *   [RU] Имитация eBPF/XDP перехвата, Kernel-Bypass буферы байт, транзит пакетов.
*   ### [📱 UE Requirements (Требования к Эмулятору Абонентов)](./technical-specification/ue-srs.md)
    *   [RU] Многопоточная генерация трафика горутинами, профили джиттера, стресс-бенчмарки.
*   ### [🛡️ PCEF Core Requirements (Требования к Исполнительному Ядру)](./technical-specification/pcef-core-srs.md)
    *   [RU] Гибридный конвейер диспетчеризации, встроенный DPI классификатор, интеграция Gy тарификации.
*   ### [⚡ Lock-Free Rate Limiter Requirements (Требования к Лимитеру)](./technical-specification/ratelimit-srs.md)
    *   [RU] 100% Lock-Free маркерная корзина (Token Bucket) на CAS-циклах с runtime.Gosched().
*   ### [📦 Reactive LRU Cache Requirements (Требования к Кэшу Сессий)](./technical-specification/lru-cache-srs.md)
    *   [RU] Алгоритм каскадного сжатия хвоста (Tail-to-Head Cascade Eviction) с ленивыми проверками.
