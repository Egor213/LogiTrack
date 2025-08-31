package grpcv1

import (
	"context"
	"errors"
	"fmt"

	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/controller/grpc/validators"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/service"
	"github.com/Egor213/LogiTrack/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogController struct {
	logService service.Log
	counters   *metrics.Counters
	loggrpc.UnimplementedLogServiceServer
}

func NewLogController(ls service.Log, cnt *metrics.Counters) *LogController {
	return &LogController{
		logService: ls,
		counters:   cnt,
	}
}

func (c *LogController) SendLog(ctx context.Context, logRequest *loggrpc.SendLogRequest) (*loggrpc.SendLogResponse, error) {
	logEntry := NewLogEntryFromRequest(logRequest)

	c.counters.GrpcRequests.Inc("SendLog", "received")
	if err := validators.Validate(logEntry); err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		logger.LogError(logEntry, err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %s", err)
	}

	// По идее одинаковые логи почти нереально создать, только если параллельный доступ и то не факт))
	logger.LogReceived(logEntry)

	id, err := c.logService.SendLog(ctx, logEntry)
	if err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		switch {
		case errors.Is(err, service.ErrLogAlreadyExists):
			logger.LogError(logEntry, err)
			return nil, status.Errorf(codes.AlreadyExists, "already exists")
		default:
			logger.LogError(logEntry, err)
			return nil, status.Errorf(codes.Unknown, "unknown error")
		}
	}

	logger.LogSaved(logEntry, id)

	c.counters.LogsReceived.Inc(logEntry.Service, logEntry.Level)
	c.counters.GrpcRequests.Inc("SendLog", "ok")

	return &loggrpc.SendLogResponse{
		LogId:  fmt.Sprint(id),
		Status: loggrpc.SendStatus_STATUS_OK,
	}, nil
}
