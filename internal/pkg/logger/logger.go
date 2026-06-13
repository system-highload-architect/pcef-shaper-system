package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

type AppLogger struct {
	fileLogger *log.Logger
}

func NewAppLogger(serviceName, logLevel string) *AppLogger {
	// Автоматически создаем папку для логов, если её нет на диске Windows/Linux
	logDir := "./logs"
	_ = os.MkdirAll(logDir, 0755)

	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, serviceName+".log"),
		MaxSize:    10,   // 10 Мегабайт
		MaxBackups: 3,    // 3 бэкапа
		Compress:   true, // Сжатие в .gz
	}

	return &AppLogger{
		fileLogger: log.New(lumberjackLogger, "["+serviceName+"] ", log.LstdFlags|log.Lmicroseconds),
	}
}

func (l *AppLogger) Info(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[INFO] %s", msg)
	l.fileLogger.Printf("[INFO] %s", msg)
}

func (l *AppLogger) Error(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[ERROR] %s", msg)
	l.fileLogger.Printf("[ERROR] %s", msg)
}

func (l *AppLogger) Fatal(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[FATAL] %s", msg)
	l.fileLogger.Printf("[FATAL] %s", msg)
	os.Exit(1)
}
