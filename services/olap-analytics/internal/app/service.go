package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"pcef-shaper-system/internal/pkg/logger"
	"pcef-shaper-system/services/olap-analytics/internal/domain"
)

type ClickHouseEmulator struct {
	mu           sync.Mutex
	dataDiskPath string
	log          *logger.AppLogger
}

func NewClickHouseEmulator(diskPath string, log *logger.AppLogger) *ClickHouseEmulator {
	_ = os.MkdirAll(diskPath, 0755)
	return &ClickHouseEmulator{
		dataDiskPath: diskPath,
		log:          log,
	}
}

// InsertBatch имитирует физику сброса пачки данных в SSTable/Partitions движка MergeTree
func (ch *ClickHouseEmulator) InsertBatch(ctx context.Context, records []*domain.AnalyticalRecord) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.log.Info("ClickHouse -> Запись пакета (Batch INSERT) из %d CDR-логов на NVMe диск...", len(records))

	// Имитируем запись в файл-сегмент СУБД ClickHouse
	segmentFile := filepath.Join(ch.dataDiskPath, "data.bin")
	f, err := os.OpenFile(segmentFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, r := range records {
		line := []byte(r.RecordID + "," + r.SubscriberID + "\n")
		_, _ = f.Write(line)
	}

	ch.log.Info("ClickHouse SUCCESS -> Пачка успешно закоммичена. Файл уплотнен (Merged).")
	return nil
}
