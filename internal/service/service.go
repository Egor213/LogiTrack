package service

import (
	"context"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
)

type Log interface {
	GetLogs(ctx context.Context, lf repotypes.LogFilter) ([]domain.LogEntry, error)
	SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error)
	GetStats(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error)
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
