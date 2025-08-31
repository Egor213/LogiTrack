package grpcv1

import (
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/domain"
)

func NewLogEntryFromRequest(req *loggrpc.SendLogRequest) *domain.LogEntry {
	return &domain.LogEntry{
		Service:   req.Service,
		Level:     req.Level.String(),
		Message:   req.Message,
		Timestamp: req.Timestamp.AsTime(),
	}
}
