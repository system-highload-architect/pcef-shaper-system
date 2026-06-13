package logger

import (
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
)

// AppLogger инкапсулирует структурированное логирование и ротацию файлов
type AppLogger struct {
	fileLogger *log.Logger
}

// NewAppLogger инициализирует кольцевой буфер логирования под лимиты K8s
func NewAppLogger(serviceName, logLevel string) *AppLogger {
	// Настраиваем ротацию lumberjack для защиты диска от переполнения
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "./logs/" + serviceName + ".log",
		MaxSize:    10,   // Максимум 10 Мегабайт на файл
		MaxBackups: 3,    // Хранить ровно 3 старых бэкапа
		Compress:   true, // Сжимать старые логи в .gz
	}

	return &AppLogger{
		fileLogger: log.New(lumberjackLogger, "["+serviceName+"] ", log.LstdFlags|log.Lmicroseconds),
	}
}

func (l *AppLogger) Info(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
	l.fileLogger.Printf("[INFO] "+format, v...)
}

func (l *AppLogger) Error(format string, v ...any) {
	log.Printf("[ERROR] "+format, v...)
	l.fileLogger.Printf("[ERROR] "+format, v...)
}

func (l *AppLogger) Fatal(format string, v ...any) {
	l.fileLogger.Printf("[FATAL] "+format, v...)
	log.Fatalf("[FATAL] "+format, v...)
}
