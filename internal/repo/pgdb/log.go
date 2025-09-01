package pgdb

import (
	"context"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/Egor213/LogiTrack/pkg/postgres"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/gommon/log"
)

type LogRepo struct {
	*postgres.Postgres
}

func NewLogRepo(pg *postgres.Postgres) *LogRepo {
	return &LogRepo{pg}
}

func (r *LogRepo) GetLogs(ctx context.Context, filter repotypes.LogFilter) ([]domain.LogEntry, error) {
	//TODO Нормально настроить фильтры, чтобы без указания уровня можно было и время корректное сделать
	conds, limit := BuildLogFilters(filter)
	log.Info("filter", filter)

	query := r.Builder.
		Select("id", "service", "level", "message", "created_at").
		From("logs").
		Limit(limit)

	if len(conds) > 0 {
		query = query.Where(sq.And(conds))
	}

	sql, args, _ := query.ToSql()
	log.Info(sql)
	rows, err := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool).Query(ctx, sql, args...)
	log.Info(rows)

	if err != nil {
		return []domain.LogEntry{}, errorsUtils.WrapPathErr(err)
	}
	defer rows.Close()

	// Рекомендуют использовать курсор, ибо это жрет память
	logs, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.LogEntry])

	if err != nil {
		return []domain.LogEntry{}, errorsUtils.WrapPathErr(err)
	}

	return logs, nil
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
		return 0, errorsUtils.WrapPathErr(err)
	}
	return id, nil
}
