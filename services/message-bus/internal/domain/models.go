package domain

import "time"

// CdrEvent описывает асинхронный лог объема трафика абонента в топике Kafka
// CdrEvent describes an asynchronous volume data log inside a Kafka topic partition
type CdrEvent struct {
	RecordID     string    `json:"record_id"`
	SubscriberID string    `json:"sub_id"`
	BytesDumped  int64     `json:"bytes_dumped"`
	Timestamp    time.Time `json:"timestamp"`
}
