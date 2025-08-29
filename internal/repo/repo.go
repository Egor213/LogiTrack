package repo

import (
	"github.com/Egor213/LogiTrack/pkg/postgres"
)

type Repositories struct {
}

func NewRepositories(pg *postgres.Postgres) *Repositories {
	return &Repositories{}
}
