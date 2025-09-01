package grpcv1

import (
	"context"
	"fmt"

	logginghelper "github.com/Egor213/LogiTrack/internal/controller/common/logging"
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/controller/grpc/validators"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/service"
	"github.com/labstack/gommon/log"
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
		logginghelper.LogError(logEntry, err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %s", err)
	}

	logginghelper.LogReceived(logEntry)

	id, err := c.logService.SendLog(ctx, logEntry)
	if err != nil {
		c.counters.GrpcRequests.Inc("SendLog", "failed")
		logginghelper.LogError(logEntry, err)
		return nil, status.Errorf(codes.Unknown, "unknown error")
	}

	logginghelper.LogSaved(logEntry, id)

	c.counters.GrpcRequests.Inc("SendLog", "ok")

	return &loggrpc.SendLogResponse{
		LogId:  fmt.Sprint(id),
		Status: loggrpc.SendStatus_STATUS_OK,
	}, nil
}

func (c *LogController) GetLogs(ctx context.Context, req *loggrpc.GetLogsRequest) (*loggrpc.GetLogsResponse, error) {
	lf := NewLogFilterFromRequest(req)
	logs, _ := c.logService.GetLogs(ctx, *lf)
	log.Info(logs)
	var temp []*loggrpc.SendLogRequest
	for _, val := range logs {
		temp = append(temp, ToSendLogRequest(val))
	}
	return &loggrpc.GetLogsResponse{
		Logs: temp,
	}, nil
}
