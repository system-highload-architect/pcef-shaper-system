# --- ПАРАМЕТРЫ СБОРКИ И ОКРУЖЕНИЯ / RUNTIME ARTIFACT MESH ---
.PHONY: all proto lint test build-all k8s-deploy-infra k8s-deploy-apps help

all: help

proto:
	@echo "⚙️ Компиляция Protobuf контрактов в Go-структуры..."
	@mkdir -p pb/gen
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       pb/*.proto
	@echo "✅ Все контракты успешно скомпилированы в pb/gen/"

lint:
	@echo "🛡️ Запуск статического анализа golangci-lint..."
	golangci-lint run ./services/... ./internal/...

test:
	@echo "⚡ Тестирование Highload контуров на наличие Race Conditions..."
	go test -v -race -timeout 30s ./services/... ./internal/...

# 4. Сборка Docker образов для всех 10 узлов монорепозитория согласно матрице go.work
# 4. Packing 10 deterministic Go docker containers aligned with active go.work setup
build-all: proto
	@echo "📦 Сборка Multi-Stage Docker контейнеров на базе Go 1.26..."
	@for service in access-gateway af-gateway message-bus ocs-rating ofcs-collector olap-analytics pcef-core pcrf-engine spr-storage ue-emulator; do \
		echo "⚙️ Сборка сжатого образа для [pcef-$$service:local]..."; \
		docker build -t pcef-$$service:local --build-arg SERVICE_PATH=services/$$service -f Dockerfile . ; \
		echo "✅ Модуль [$$service] успешно упакован."; \
	done
	@echo "🎉 Сборка всех 10 образов завершена! Слой виртуализации готов."

k8s-deploy-infra:
	@echo "🏛️ Развертывание тяжелой инфраструктуры в Kubernetes..."
	kubectl apply -f k8s/pcef-configmap.yaml
	kubectl apply -f k8s/infra/
	@echo "⏳ Базы данных и очереди запущены."

k8s-deploy-apps: build-all
	@echo "🚀 Накатывание 7 Go-микросервисов в рантайм Kubernetes..."
	kubectl apply -f k8s/apps/
	@echo "🎉 Весь распределенный 3GPP PCC контур успешно развернут в K8s кластере!"

help:
	@echo "🏛️ ПУЛЬТ УПРАВЛЕНИЯ PCEF SHAPER CLUSTER (GO WORKSPACES 1.26):"
	@echo "  make proto             - Скомпилировать Protobuf контракты"
	@echo "  make lint              - Запустить линтеры"
	@echo "  make test              - Запустить юнит-тесты на Race Conditions"
	@echo "  make build-all         - Собрать Multi-Stage Docker образы для 10 сервисов"
	@echo "  make k8s-deploy-infra  - Развернуть СУБД и Брокеры в K8s"
	@echo "  make k8s-deploy-apps   - Скомпилировать и раскатать 7 Go-сервисов в K8s"
