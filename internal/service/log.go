package service

import (
	"context"
	"errors"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repoerrs"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
)

type LogService struct {
	logRepo  repo.Log
	counters *metrics.Counters
}

func NewLogService(lr repo.Log, cnt *metrics.Counters) *LogService {
	return &LogService{
		logRepo:  lr,
		counters: cnt,
	}
}

func (s *LogService) GetLogs() int {
	return 1
}

func (s *LogService) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	id, err := s.logRepo.SendLog(ctx, logObj)
	if err != nil {
		if errors.Is(err, repoerrs.ErrAlreadyExists) {
			return 0, ErrLogAlreadyExists
		}
		return 0, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return id, nil
}

func (s *LogService) GetStats() int {
	return 1
}
