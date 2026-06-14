package app

import "context"

// CdrPipelineОrchestrator определяет b2b-контракт асинхронного выгребания и пакетного сброса логов
type CdrPipelineOrchestrator interface {
	StartPipeline(ctx context.Context)
	Stop()
}
