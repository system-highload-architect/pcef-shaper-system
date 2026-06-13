package grpc

import (
	"io"
	"log"

	gen "pcef-shaper-system/pb/gen"
	"pcef-shaper-system/services/pcef-core/internal/app"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	gen.UnimplementedTrafficPipelineServer
	service app.ShaperEngine
}

func NewGrpcHandler(service app.ShaperEngine) *GrpcHandler {
	return &GrpcHandler{service: service}
}

// ProcessTrafficStream — Full-Duplex бинарный поток HTTP/2 для наносекундного шейпинга
func (h *GrpcHandler) ProcessTrafficStream(stream gen.TrafficPipeline_ProcessTrafficStreamServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Считываем фрейм сетевого пакета из сокета шлюза
			frame, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return status.Errorf(codes.DataLoss, "Failed to read network frame socket: %v", err)
			}

			// Прогоняем пакет сквозь плоский конвейер ядра
			verdict, err := h.service.ProcessPacket(ctx, frame)
			if err != nil {
				log.Printf("[PCEF ERROR] Ошибка конвейера диспетчеризации: %v", err)
				continue
			}

			// Атомарно бомбардируем шлюз ответным вердиктом в реальном времени
			if err := stream.Send(verdict); err != nil {
				return status.Errorf(codes.Unavailable, "Failed to send enforcement verdict: %v", err)
			}
		}
	}
}
