package pgdb

import (
	"context"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/repoerrs"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/Egor213/LogiTrack/pkg/postgres"
)

type LogRepo struct {
	*postgres.Postgres
}

func NewLogRepo(pg *postgres.Postgres) *LogRepo {
	return &LogRepo{pg}
}

func (r *LogRepo) GetLogs(ctx context.Context, filter repotypes.LogFilterInput) ([]*domain.LogEntry, error) {
	return nil, nil
}

func (r *LogRepo) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	sql, args, _ := r.Builder.
		Insert("logs").
		Columns("service", "level", "message").
		Values(logObj.Service, logObj.Level, logObj.Message).
		Suffix("RETURNING id").
		ToSql()
	var id int
	err := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool).QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		if errorsUtils.IsUniqueViolation(err) {
			return 0, repoerrs.ErrAlreadyExists
		}
		return 0, errorsUtils.WrapPathErr(err)
	}
	return id, nil
}
