# 🗺️ PCEF Architecture Navigation Index / Индекс навигации по архитектуре PCEF

[RU] Этот документ является единой точкой навигации по низкоуровневой архитектуре эшелонов Policy and Charging Control (PCC). Каждый компонент изолирован в отдельную спецификацию, описывающую внутреннее устройство, стейт-машины, утилизацию ресурсов и протоколы взаимодействия.

[EN] This document serves as a unified navigation index for the low-level Policy and Charging Control (PCC) architecture layers. Each component is isolated into a separate specification detailing its internal mechanics, state machines, resource utilization, and interaction protocols.

---

## 📦 Control Plane Components / Компоненты плоскости управления

*   ### [🚀 Application Function (AF)](./specification/af-specification.md)
    *   [RU] Управление контентом динамических b2b-сессий, SLA-запросы, интерфейс Rx (Diameter).
    *   [EN] Dynamic b2b session content management, SLA enforcement requests, Rx interface (Diameter).
*   ### [⚙️ Policy & Charging Rules Function (PCRF)](./specification/pcrf-specification.md)
    *   [RU] Центральное ядро бизнес-логики, компиляция PCC-правил, оркестрация интерфейсов Gx/Sp.
    *   [EN] Central business logic core, PCC rules compilation, Gx/Sp interfaces orchestration.
*   ### [🗄️ Subscription Profile Repository (SPR / UDR)](./specification/spr-specification.md)
    *   [RU] Хранилище профилей абонентов, тарифов и b2b-лимитов, оптимизация схем данных.
    *   [EN] Subscriber profiles, tariffs, and b2b limits repository, data schema optimization.
*   ### [💸 Online Charging System (OCS)](./specification/ocs-specification.md)
    *   [RU] Система реального времени тарификации, квантование трафика (Quota Reservation), интерфейс Gy.
    *   [EN] Real-time online rating engine, traffic quantization (Quota Reservation), Gy interface.
*   ### [📊 Offline Charging System (OFCS)](./specification/ofcs-specification.md)
    *   [RU] Система отложенного биллинга, CDR-сборщики (Call Detail Records), экспорт в ClickHouse, интерфейс Gz.
    *   [EN] Post-paid billing engine, CDR (Call Detail Records) collectors, ClickHouse export, Gz interface.
*   ### [🔒 RADIUS / AAA Server](./specification/aaa-specification.md)
    *   [RU] Аутентификация, авторизация и аккаунтинг сессий, управление пулами IP-адресов.
    *   [EN] Authentication, authorization, and session accounting, IP pool management.

---

## 🟢 User Plane Components / Компоненты плоскости пользователя

*   ### [📱 User Equipment (UE)](./specification/ue-specification.md)
    *   [RU] Терминалы абонентов, генерация L4-L7 трафика, джиттер и сетевые профили нагрузок.
    *   [EN] Subscriber end-user terminals, L4-L7 traffic generation, jitter, and network load profiling.
*   ### [🌐 Access Gateway (BNG / PGW / UPF)](./specification/gateway-specification.md)
    *   [RU] Шлюзы терминации трафика, DPDK/eBPF магистрали, RADIUS-сигнализация UDP (1812/1813).
    *   [EN] Traffic termination gateways, DPDK/eBPF pipelines, RADIUS UDP signaling (1812/1813).
