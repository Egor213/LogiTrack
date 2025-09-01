package pgdb

import (
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	sq "github.com/Masterminds/squirrel"
)

func BuildLogFilters(filter repotypes.LogFilter) ([]sq.Sqlizer, uint64) {
	conds := []sq.Sqlizer{}

	if filter.Service != "" {
		conds = append(conds, sq.Eq{"service": filter.Service})
	}
	if filter.Level != "" {
		conds = append(conds, sq.Eq{"level": filter.Level})
	}
	if !filter.From.IsZero() {
		conds = append(conds, sq.GtOrEq{"created_at": filter.From})
	}
	if !filter.To.IsZero() {
		conds = append(conds, sq.LtOrEq{"created_at": filter.To})
	}

	limit := uint64(100)
	if filter.Limit > 0 {
		limit = uint64(filter.Limit)
	}

	return conds, limit
}
