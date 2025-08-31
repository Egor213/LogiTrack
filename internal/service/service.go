package service

import (
	"context"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
)

type Log interface {
	GetLogs() int
	SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error)
	GetStats() int
}

type Services struct {
	Log
}

type ServicesDependencies struct {
	Repos    *repo.Repositories
	Counters *metrics.Counters
}

func NewServices(deps ServicesDependencies) *Services {
	return &Services{
		Log: NewLogService(deps.Repos.Log, deps.Counters),
	}
}
