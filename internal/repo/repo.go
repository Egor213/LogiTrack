package repo

import (
	"context"
	"time"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/pgdb"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	"github.com/Egor213/LogiTrack/pkg/postgres"
)

type Log interface {
	GetLogs(ctx context.Context, filter repotypes.LogFilter) ([]domain.LogEntry, error)
	SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error)
	GetStatsByService(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error)
}

type Repositories struct {
	Log
}

func NewRepositories(pg *postgres.Postgres) *Repositories {
	return &Repositories{
		Log: pgdb.NewLogRepo(pg),
	}
}
