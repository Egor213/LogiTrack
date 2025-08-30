package grpcv1

import (
	"context"

	"github.com/Egor213/LogiTrack/internal/metrics"

	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
)

type LogController struct {
	loggrpc.UnimplementedLogServiceServer
}

func NewLogController() *LogController {
	return &LogController{}
}

func (c *LogController) SendLog(ctx context.Context, logRequest *loggrpc.LogEntry) (*loggrpc.SendLogResponse, error) {
	// TODO: Тут надо кафку добавить
	// TODO: Прометеус настроить тут
	temp := metrics.New()
	temp.LogsPublished.Inc("Test", "DEBUG")
	return &loggrpc.SendLogResponse{
		LogId:  "1",
		Status: 12,
	}, nil
}
