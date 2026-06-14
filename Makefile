# ... (все предыдущие цели proto, lint, test, build-all, k8s остаются без изменений)

.PHONY: run-local stop-local

# 7. Локальный сквозной запуск всех 10 компонентов кластера в фоне одного терминала
# 7. Spawning all 10 network nodes concurrently as independent OS background processes
run-local: stop-local
	@echo "🚀 [START] Запуск слоя СУБД, Брокеров и Кэш-индексов (In-Memory Эмуляторы)..."
	@go run services/spr-storage/cmd/main.go &
	@go run services/message-bus/cmd/main.go &
	@go run services/olap-analytics/cmd/main.go &
	@go run services/ocs-rating/cmd/main.go &
	@sleep 2
	@echo "⚙️ [START] Запуск слоя Сигнализации и Управления (Control Plane)..."
	@go run services/pcrf-engine/cmd/main.go &
	@go run services/af-gateway/cmd/main.go &
	@go run services/ofcs-collector/cmd/main.go &
	@sleep 2
	@echo "🛡️ [START] Запуск Исполнительного Ядра и Сетевого Шлюза (User Plane)..."
	@go run services/pcef-core/cmd/main.go &
	@go run services/access-gateway/cmd/main.go &
	@sleep 2
	@echo "🔥 [START] Спусковой крючок: Включение нагрузочногоue-emulator смартфона..."
	@go run services/ue-emulator/cmd/main.go

# 8. Экстренное принудительное завершение всех фоновых Go процессов кластера
# 8. Forcefully terminating all rogue background Go processes running in the OS space
stop-local:
	@echo "🛑 Остановка всех фоновых Go-сервисов и очистка сетевых сокетов..."
	@pkill -f "services/" || true
	@echo "✅ Все локальные процессы успешно зачищены."

# Обновляем справочное меню help, добавляя новые команды
help:
	@echo "🏛️ ПУЛЬТ УПРАВЛЕНИЯ PCEF SHAPER CLUSTER (GO WORKSPACES 1.26):"
	@echo "  make proto             - Скомпилировать Protobuf контракты"
	@echo "  make lint              - Запустить линтеры"
	@echo "  make test              - Запустить юнит-тесты на Race Conditions"
	@echo "  make build-all         - Собрать Multi-Stage Docker образы для 10 сервисов"
	@echo "  make k8s-deploy-infra  - Развернуть СУБД и Брокеры в K8s"
	@echo "  make k8s-deploy-apps   - Скомпилировать и раскатать Go-сервисы в K8s"
	@echo "  make run-local         - Локальный запуск всех 10 сервисов в фоне"
	@echo "  make stop-local        - Принудительно убить все фоновые процессы сервисов"
