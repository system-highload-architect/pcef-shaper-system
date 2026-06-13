package app

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"pcef-shaper-system/internal/pkg/logger" // Напрямую импортируем наше шасси!
	gen "pcef-shaper-system/pb/gen"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type TrafficGenerator struct {
	clientCount int
	pipelineCli gen.TrafficPipelineClient
	stopChan    chan struct{}
	wg          sync.WaitGroup
	log         *logger.AppLogger // Используем единый тип логера
}

func NewTrafficGenerator(clientCount int, pipelineCli gen.TrafficPipelineClient, log *logger.AppLogger) *TrafficGenerator {
	return &TrafficGenerator{
		clientCount: clientCount,
		pipelineCli: pipelineCli,
		stopChan:    make(chan struct{}),
		log:         log,
	}
}

func (g *TrafficGenerator) StartLoadTest(ctx context.Context) error {
	g.log.Info("Запуск Highload стресс-теста: %d горутин-абонентов", g.clientCount)

	hosts := []string{"youtube.com", "netflix.com", "telegram.org", "whatsapp.com", "unknown-hack-site.ru"}

	for i := 1; i <= g.clientCount; i++ {
		g.wg.Add(1)
		imsi := fmt.Sprintf("2500100000000%02d", (i%2)+1)
		ip := fmt.Sprintf("192.168.1.%d", 49+i)

		go g.spawnDeviceWorker(ctx, imsi, ip, hosts)
	}
	return nil
}

func (g *TrafficGenerator) spawnDeviceWorker(ctx context.Context, imsi, ip string, hosts []string) {
	defer g.wg.Done()

	stream, err := g.pipelineCli.ProcessTrafficStream(ctx)
	if err != nil {
		g.log.Error("[%s] Не удалось открыть gRPC сокет потока трафика: %v", imsi, err)
		return
	}
	defer stream.CloseSend()

	go func() {
		for {
			verdict, err := stream.Recv()
			if err == io.EOF || err != nil {
				return
			}
			g.log.Info("SHAPER VERDICT -> IP: %s | Действие WAF: %s | Скорость: %d бит/с",
				verdict.SourceIp, verdict.Action, verdict.AllowedBandwidth)
		}
	}()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-g.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			payloadSize := int64(r.Intn(1.5 * 1024 * 1024))
			targetHost := hosts[r.Intn(len(hosts))]

			err := stream.Send(&gen.RawPacketFrame{
				SourceIp:         ip,
				DestinationHost:  targetHost,
				PayloadSizeBytes: payloadSize,
				Timestamp:        timestamppb.Now(),
			})
			if err != nil {
				return
			}

			time.Sleep(time.Duration(100+r.Intn(400)) * time.Millisecond)
		}
	}
}

func (g *TrafficGenerator) StopLoadTest() {
	close(g.stopChan)
	g.wg.Wait()
	g.log.Info("Нагрузочный генератор успешно остановлен.")
}
