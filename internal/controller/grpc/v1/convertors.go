package grpcv1

import (
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewLogEntryFromRequest(req *loggrpc.SendLogRequest) *domain.LogEntry {
	return &domain.LogEntry{
		Service:   req.Service,
		Level:     req.Level.String(),
		Message:   req.Message,
		Timestamp: req.Timestamp.AsTime(),
	}
}

func NewLogFilterFromRequest(req *loggrpc.GetLogsRequest) *repotypes.LogFilter {
	return &repotypes.LogFilter{
		Service: req.Service,
		Level:   req.Level.String(),
		From:    req.From.AsTime(),
		To:      req.To.AsTime(),
		Limit:   int(req.Limit),
	}
}

func ToSendLogRequest(entry domain.LogEntry) *loggrpc.SendLogRequest {
	var level loggrpc.LogLevel
	switch entry.Level {
	case "INFO":
		level = loggrpc.LogLevel_INFO
	case "WARN":
		level = loggrpc.LogLevel_WARN
	case "ERROR":
		level = loggrpc.LogLevel_ERROR
	default:
		level = loggrpc.LogLevel_LOG_LEVEL_UNSPECIFIED
	}

	return &loggrpc.SendLogRequest{
		Service:   entry.Service,
		Level:     level,
		Message:   entry.Message,
		Timestamp: timestamppb.New(entry.Timestamp),
	}
}
