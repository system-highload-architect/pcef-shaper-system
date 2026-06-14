# 🗺️ PCEF Architecture & SRS Navigation Index / Индекс базы знаний PCEF

[RU] Этот документ является единой точкой навигации по низкоуровневой архитектуре, диаграммам последовательностей и техническим требованиям (SRS) системы Policy and Charging Control (PCC).

[EN] This document serves as a unified navigation index for the low-level architecture, sequence charts, and Technical Requirements Specifications (SRS) of the PCC system.

---

## 📦 Component Specifications & Mermaid Diagrams / Архитектура и Диаграммы

*   ### [🚀 Application Function (AF)](./specification/af-specification.md) — Сигнализация Diameter Rx.
*   ### [⚙️ Policy & Charging Rules Function (PCRF)](./specification/pcrf-specification.md) — Движок политик Gx/Sp.
*   ### [🗄️ Subscription Profile Repository (SPR)](./specification/spr-specification.md) — База профилей ScyllaDB.
*   ### [💸 Online Charging System (OCS)](./specification/ocs-specification.md) — Квантование Gy в Aerospike.
*   ### [📊 Offline Charging System (OFCS)](./specification/ofcs-specification.md) — Асинхронные CDR логи в Kafka/ClickHouse.
*   ### [🔒 RADIUS / AAA Server](./specification/aaa-specification.md) — UDP перехват сессий.
*   ### [🌐 Access Gateway (BNG / UPF)](./specification/gateway-specification.md) — eBPF/XDP Kernel Bypass.
*   ### [📱 User Equipment (UE)](./specification/ue-specification.md) — Высоконагруженный генератор трафика.

---