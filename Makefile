# --- ПАРАМЕТРЫ СБОРКИ И ОКРУЖЕНИЯ / ARTIFACT ENVELOPE CONFIG ---
.PHONY: all proto lint test build-all k8s-deploy-infra k8s-deploy-apps help

all: help

# 1. Генерация Protobuf-контрактов для всех интерфейсов 3GPP PCC
# 1. Spawning typed Go code generation from all 3GPP PCC proto definitions
proto:
	@echo "⚙️ Компиляция Protobuf контрактов в Go-структуры..."
	@mkdir -p pb/gen
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pb/*.proto
	@echo "✅ Все контракты успешно скомпилированы в pb/gen/"

# 2. Статический анализ кодовой базы на антипаттерны и утечки
# 2. Static linting execution checking code for anomalies and goroutine leaks
lint:
	@echo "🛡️ Запуск статического анализа golangci-lint..."
	golangci-lint run ./services/... ./internal/...

# 3. Запуск юнит-тестов с жестким профайлингом скрытых гонок данных
# 3. Unit-testing suite execution with aggressive race condition tracking
test:
	@echo "⚡ Тестирование Highload контуров на наличие Race Conditions..."
	go test -v -race -timeout 30s ./services/... ./internal/...

# 4. Сборка Docker образов для всех 7 микросервисов в локальный кэш
# 4. Building local Multi-Stage Docker images for all 7 independent services
build-all: proto
	@echo "📦 Сборка Docker контейнеров для Go-микросервисов..."
	@for service in access-gateway af-gateway ocs-rating ofcs-collector pcef-core pcrf-engine ue-emulator; do \
		echo "⚙️ Сборка образа для [$$service]..."; \
		docker build -t $$service:local -f services/$$service/Dockerfile . ; \
	 Eagle \
	done
	@echo "✅ Сборка всех 7 образов завершена успешно!"

# 5. Развертывание 3-х узлов инфраструктуры баз данных и очередей в Kubernetes
# 5. Spawning 3 heavy infrastructure stateful nodes into active Kubernetes namespace
k8s-deploy-infra:
	@echo "🏛️ Развертывание тяжелой инфраструктуры (ScyllaDB, Kafka, ClickHouse)..."
	kubectl apply -f k8s/pcef-configmap.yaml
	kubectl apply -f k8s/infra/
	@echo "⏳ Инфраструктура создана. Проверьте статус подов через: kubectl get pods"

# 6. Развертывание 7 Go-микросервисов в рантайм Кубера
# 6. Spawning 7 custom Go-microservices into active Kubernetes cluster mesh
k8s-deploy-apps: build-all
	@echo "🚀 Накатывание 7 Go-микросервисов в кластер Kubernetes..."
	kubectl apply -f k8s/apps/
	@echo "🎉 Весь распределенный 3GPP PCC контур успешно развернут в K8s!"

help:
	@echo "🏛️ ПУЛЬТ УПРАВЛЕНИЯ PCEF SHAPER CLUSTER MAKEFILE:"
	@echo "  make proto             - Скомпилировать Protobuf контракты"
	@echo "  make lint              - Запустить гоу линтеры"
	@echo "  make test              - Запустить тесты с флагом -race"
	@echo "  make build-all         - Собрать Docker образы для всех сервисов"
	@echo "  make k8s-deploy-infra  - Развернуть СУБД и Брокеры в K8s"
	@echo "  make k8s-deploy-apps   - Скомпилировать и раскатать 7 Go-сервисов в K8s"
