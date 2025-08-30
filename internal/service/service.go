package service

import (
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
)

type Services struct {
}

type ServicesDependencies struct {
	Repos    *repo.Repositories
	Counters *metrics.Counters
}

func NewServices(deps ServicesDependencies) *Services {
	return &Services{}
}
