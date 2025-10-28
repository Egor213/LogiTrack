package pgdb

import (
	"time"

	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	sq "github.com/Masterminds/squirrel"
)

func BuildLogQueryFilters(filter repotypes.LogFilter) ([]sq.Sqlizer, uint64) {
	conds := []sq.Sqlizer{}

	if filter.Service != "" {
		conds = append(conds, sq.Eq{"service": filter.Service})
	}

	if filter.Level != "" && filter.Level != "LOG_LEVEL_UNSPECIFIED" {
		conds = append(conds, sq.Eq{"level": filter.Level})
	}
	if !filter.From.IsZero() && !filter.From.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.GtOrEq{"created_at": filter.From})
	}
	if !filter.To.IsZero() && !filter.From.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.LtOrEq{"created_at": filter.To})
	}

	limit := uint64(100)
	if filter.Limit > 0 {
		limit = uint64(filter.Limit)
	}

	return conds, limit
}

func BuildServiceStatsQueryFilters(service string, from, to time.Time) []sq.Sqlizer {
	conds := []sq.Sqlizer{}
	if service != "" {
		conds = append(conds, sq.Eq{"service": service})
	}

	if !from.IsZero() && !from.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.GtOrEq{"created_at": from})
	}
	if !to.IsZero() && !from.Equal(time.Unix(0, 0)) {
		conds = append(conds, sq.LtOrEq{"created_at": to})
	}
	return conds
}
