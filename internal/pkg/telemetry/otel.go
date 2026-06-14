package telemetry

import (
	"net/http"
	"pcef-shaper-system/internal/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type OtelMetrics struct {
	meter          metric.Meter
	BlockedPackets metric.Int64Counter // Счетчик атак, заблокированных Rate-Limiter'ом
	ProcessedBytes metric.Int64Counter // Счетчик успешно прогнанных байт трафика
}

// InitOtelMetrics взводит вендоро-независимый мониторинг и поднимает HTTP сокет :8080/metrics
func InitOtelMetrics(serviceName string, bindMetricsAddr string, log *logger.AppLogger) (*OtelMetrics, error) {
	// 1. Инициализируем OTel экспортёр в формате Prometheus Scraping
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// 2. Взводим SDK-провайдер метрик
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	meter := provider.Meter(serviceName)

	// 3. Создаем строго типизированные b2b метрики по ТЗ
	blocked, _ := meter.Int64Counter(
		"pcef_blocked_packets_total",
		metric.WithDescription("Общее количество вредоносных пакетов, уничтоженных эшелоном Rate-Limiter/WAF посредством XDP_DROP"),
	)

	processed, _ := meter.Int64Counter(
		"pcef_processed_bytes_total",
		metric.WithDescription("Общий объем легитимного трафика пользователя, успешно пропущенного сквозь QoS-шейпер скорости"),
	)

	// 4. Нативно запускаем изолированный HTTP-сервер для Prometheus-клиентов в фоне
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler()) // Прометеус будет забирать метрики отсюда
		log.Info("Observability Core -> HTTP-порт сбора OpenTelemetry/Prometheus метрик успешно запущен на %s/metrics", bindMetricsAddr)
		if err := http.ListenAndServe(bindMetricsAddr, mux); err != nil {
			log.Error("Сбой HTTP-сервера метрик: %v", err)
		}
	}()

	return &OtelMetrics{
		meter:          meter,
		BlockedPackets: blocked,
		ProcessedBytes: processed,
	}, nil
}
