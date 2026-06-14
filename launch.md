# 🚀 PCEF Shaper System Cluster Launch Guide / Руководство по запуску кластера

[RU] Настоящий документ содержит исчерпывающие инструкции по сквозному запуску, тестированию и остановке всех 10 компонентов распределенной b2b-платформы как в локальном режиме (без инфраструктурного оверхеда), так и внутри Kubernetes кластера.

[EN] This document provides exhaustive blueprints for end-to-end execution, capacity testing, and graceful teardown of all 10 distributed platform nodes both in local background mode and inside a Kubernetes namespace.

---

## ⚡ 1. Local Monolithic Launch / Локальный скоростной запуск (Для дебага и Code Review)

[RU] Для проверки рантайма, Lock-Free CAS вычислений, каскадного LRU-кэша и прохождения трафика без установки Kubernetes, все 10 узлов запускаются параллельно в виде фоновых процессов ОС внутри одного терминала одной командой `Makefile`.

[EN] To verify hot-path execution, Lock-Free CAS mutations, David's cascade LRU eviction, and overall traffic streams without a Kubernetes overhead, all 10 nodes can be spawned concurrently as background OS processes within a single terminal using a unified `Makefile` command.

### 🛠️ Инструкция по запуску / Execution Steps:
1. Откройте **один терминал Git Bash** в корневой директории проекта `pcef-shaper-system/`.
2. Запустите автоматизированный конвейер сборки контрактов и параллельного старта всех слоев:
   ```bash
   make run-local
   ```
3. Система последовательно проинициализирует 4 In-Memory хранилища данных, взведет слои Control Plane сигналов, запустит User Plane ядро и включит многопоточный `ue-emulator` на горутинах [🧠].

### 📊 Анализ бегущего лога в консоли / Stream Telemetry Logs Architecture:
При успешном запуске в терминал польется монолитный, наносекундный b2b-водопад логов взаимодействия:
* **`[ocs-rating-engine] [INFO] RPC SUCCESS`**: подтверждает, что gRPC-интерцептор трассировки фиксирует задержку `Latency: 0s` благодаря Lock-Free CAS-циклам процессора без Mutex Contention [🧠].
* **`[pcef-core-engine] [INFO] SHAPER VERDICT -> IP: ... | ALLOW`**: подтверждает, что прилетел RADIUS сигнальный триггер, абонент авторизовался в кэше, битовые маски успешно сопоставили предикаты за $O(1)$ и выделили QoS-скорость в 50 Mbps для YouTube [🧠].
* **`[pcef-core-engine] [INFO] SHAPER VERDICT -> IP: ... | XDP_DROP`**: подтверждает, что L7-лимитер маркерной корзины или защитный периметр ядра перехватили неавторизованный IP-адрес/флуд и мгновенно уничтожили пакет на уровне сетевого драйвера со скоростью 0 бит/с, спасая RAM и CPU от выжигания тактов [🧠].

### 🛑 Безопасная остановка кластера / Graceful Teardown:
Чтобы плавно остановить все фоновые процессы, закройте gRPC-каналы и вернуть страницы памяти Go-кучи операционной системе, нажмите в окне терминала:
```bash
Ctrl + C
```
*Если процессы зависли в фоне ОС, выполните команду принудительной зачистки сокетов:*
```bash
make stop-local
```

---

## 📊 2. Live Telemetry Metric Monitoring / Проверка Офлайнового Мониторинга (OTel)

[RU] Пока локальный кластер крутится под Highload-нагрузкой генератора, вы можете проверить готовность системы к сдаче на прод (*Observability Ready*). Наше общее шасси поднимает изолированный HTTP-сервер для Prometheus-агентов.

[EN] While the local cluster functions under intensive load-generation streams, you can audit the platform's production telemetry layer. Our shared infrastructure chassis provisions an isolated HTTP metric exporter endpoint.

* Откройте любой браузер на своем компьютере и перейдите по адресу:
  👉 **`http://localhost:8080/metrics`**

Вы увидите структурированные b2b-счетчики OpenTelemetry API, непрерывно растущие в реальном времени:
1. `pcef_processed_bytes_total` — суммарный объем легитимных мегабайт, пропущенных QoS-шейпером скорости [🧠].
2. `pcef_blocked_packets_total` — количество вредоносных DDoS-пакетов флуда, успешно уничтоженных лимитером частоты на CAS-атомиках посредством `XDP_DROP` [🧠].

---

## 🏗️ 3. Industrial Kubernetes Deploy / Развертывание в Продакшен-кластер K8s

[RU] Для промышленной раскатки 10 автономных узлов с правилами балансировки Round-Robin, лимитами ресурсов CPU/RAM и автоскейлингом HPA до 3+ копий, используется оркестрация Kubernetes (Kind/Minikube).

[EN] For production-grade deployment featuring Round-Robin load balancing, explicit CPU/RAM resource constraints, and dynamic HPA auto-scaling up to 3+ instances under peak stress, Kubernetes orchestration manifests are provided.

```bash
# 1. Компиляция и сборка ультра-легковесных Multi-Stage Docker-образов (по ~18 MB каждый)
make build-all

# 2. Развертывание тяжелой инфраструктуры (StatefulSet ScyllaDB, ClickHouse, шина Kafka)
make k8s-deploy-infra

# 3. Накатывание 7 Go-микросервисов с инъекцией переменных из общего ConfigMap через CoreDNS K8s
make k8s-deploy-apps
```
