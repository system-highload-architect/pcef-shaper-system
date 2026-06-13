package grpc

import (
	"io"

	"pcef-shaper-system/internal/pkg/logger" // Подключаем наше общее платформенное шасси / Shared chassis logger
	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedTrafficPipelineServer
	service app.ShaperEngine
	log     *logger.AppLogger // Единый структурированный логер платформы / Centralized app logger
}

func NewGrpcHandler(service app.ShaperEngine, log *logger.AppLogger) *GrpcHandler {
	return &GrpcHandler{
		service: service,
		log:     log,
	}
}

// ProcessTrafficStream — Full-Duplex бинарный поток HTTP/2 для наносекундного шейпинга трафика
// ProcessTrafficStream — Full-Duplex binary HTTP/2 stream for nanosecond-level packet traffic shaping
func (h *GrpcHandler) ProcessTrafficStream(stream gen.TrafficPipeline_ProcessTrafficStreamServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Считываем входящий бинарный фрейм сетевого пакета из сокета шлюза доступа
			// Ingesting incoming binary network packet frames directly from the gateway socket
			frame, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return status.Errorf(codes.DataLoss, "Failed to read network frame socket: %v", err)
			}

			// Прогоняем сетевой фрейм сквозь гибридный плоский конвейер ядра PCEF
			// Routing the ingested packet frame through the flat hybrid PCEF shaper pipeline
			verdict, err := h.service.ProcessPacket(ctx, frame)
			if err != nil {
				// Логируем инцидент нарушения/сбоя через платформенный логер в файл ротации
				// Writing shaper processing anomalies directly into rolling lumberjack files
				h.log.Error("Ошибка конвейера диспетчеризации PCEF: %v", err)
				continue
			}

			// Атомарно бомбардируем шлюз ответным вердиктом применения QoS-политики в реальном времени
			// Atomically transmitting enforcement QoS verdicts back to the gateway in real-time
			if err := stream.Send(verdict); err != nil {
				return status.Errorf(codes.Unavailable, "Failed to send enforcement verdict: %v", err)
			}
		}
	}
}
