package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type LoggerInterface interface {
	Info(format string, v ...any)
}

type GracefulServer interface {
	GracefulStop()
}

// ListenSignals блокирует горутину, ожидая системных команд ядра Linux (SIGINT, SIGTERM)
func ListenSignals(log LoggerInterface, server GracefulServer, timeout time.Duration) {
	sigChan := make(chan os.Signal, 1)
	// Перехватываем сигналы корректного завершения от Kubernetes/ОС
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Блокировка до прилета системного фрейма
	sig := <-sigChan
	log.Info("Получен системный сигнал ядра Linux [%s]. Инициирован Graceful Shutdown...", sig.String())

	// Выделяем сервису жесткий b2b-таймаут на завершение транзакций в RAM
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stopChan := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopChan)
	}()

	select {
	case <-stopChan:
		log.Info("Все сетевые сокеты и gRPC соединения успешно закрыты. Сервер остановлен штатно.")
	case <-ctx.Done():
		log.Info("Превышен таймаут ожидания завершения транзакций. Принудительная остановка процесса.")
	}
}
