package service

import (
	"context"

	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	"github.com/labstack/gommon/log"
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

func (s *LogService) GetLogs(ctx context.Context, lf repotypes.LogFilter) ([]domain.LogEntry, error) {
	logs, err := s.logRepo.GetLogs(ctx, lf)
	log.Info(logs, err)
	if err != nil {
		return []domain.LogEntry{}, err
	}
	return logs, nil
}

func (s *LogService) SendLog(ctx context.Context, logObj *domain.LogEntry) (int, error) {
	s.counters.LogsReceived.Inc(logObj.Service, logObj.Level)
	id, err := s.logRepo.SendLog(ctx, logObj)
	if err != nil {
		return 0, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return id, nil
}

func (s *LogService) GetStats() int {
	return 1
}
