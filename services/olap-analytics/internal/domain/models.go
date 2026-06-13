package domain

import "time"

// AnalyticalRecord описывает колоночную структуру CDR лога на диске
type AnalyticalRecord struct {
	RecordID     string    `json:"record_id"`
	SubscriberID string    `json:"subscriber_id"`
	BytesDumped  int64     `json:"bytes_dumped"`
	Timestamp    time.Time `json:"timestamp"`
}
