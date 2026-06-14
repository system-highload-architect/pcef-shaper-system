# =================================================================
# STAGE 1: LOCK-FREE COMPILATION ENGINE (BUILD STAGE)
# =================================================================
# Используем строго версию Go 1.26 для нативной поддержки воркспейса
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git gcc musl-dev make

WORKDIR /app

# 1. Сначала копируем управляющую матрицу go.work и общие платформенные модули
COPY go.work ./
COPY internal/ ./internal/
COPY pb/ ./pb/

# 2. Копируем исходный код ВСЕХ 10 модулей, зарегистрированных вuse блоке go.work
# Это гарантирует, что внутренний линкер Go 1.26 безошибочно соберет локальные зависимости
COPY services/access-gateway/ ./services/access-gateway/
COPY services/af-gateway/ ./services/af-gateway/
COPY services/message-bus/ ./services/message-bus/
COPY services/ocs-rating/ ./services/ocs-rating/
COPY services/ofcs-collector/ ./services/ofcs-collector/
COPY services/olap-analytics/ ./services/olap-analytics/
COPY services/pcef-core/ ./services/pcef-core/
COPY services/pcrf-engine/ ./services/pcrf-engine/
COPY services/spr-storage/ ./services/spr-storage/
COPY services/ue-emulator/ ./services/ue-emulator/

# 3. Принимаем аргумент пути конкретного целевого сервиса для b2b-изоляции
ARG SERVICE_PATH

# Переходим в рабочую директорию целевого модуля и подтягиваем внешние библиотеки
WORKDIR /app/${SERVICE_PATH}
RUN go mod download

# Сжимаем бинарник статической линковкой, вырезая отладочную символику dsym флагами -s -w
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/service ./cmd/main.go

# =================================================================
# STAGE 2: ULTRA-LIGHTWEIGHT PRODUCTION ENVELOPE (RUN STAGE)
# =================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /root/

# Забираем чистый оптимизированный бинарный артефакт из builder эшелона
COPY --from=builder /app/service ./service

# Динамически прокидываем локальный config.yaml для тестов в Kind кластере
ARG SERVICE_PATH
COPY ${SERVICE_PATH}/config.yaml ./config.yaml

# Экспортируем канонический b2b-пул портов
EXPOSE 50050 50052 50054 9042 9092 8123

CMD ["./service"]
