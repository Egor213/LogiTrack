package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Egor213/LogiTrack/internal/broker"
	"github.com/Egor213/LogiTrack/internal/domain"
	"github.com/Egor213/LogiTrack/internal/metrics"
	"github.com/Egor213/LogiTrack/internal/repo"
	"github.com/Egor213/LogiTrack/internal/repo/repotypes"
	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type LogService struct {
	logRepo        repo.Log
	counters       *metrics.Counters
	brokerProducer broker.Producer
}

func NewLogService(lr repo.Log, cnt *metrics.Counters, p broker.Producer) *LogService {
	return &LogService{
		logRepo:        lr,
		counters:       cnt,
		brokerProducer: p,
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
	err := s.brokerProducer.SendMessage(ctx, []byte(fmt.Sprintf("service=%s, level=%s, Message=%s", logObj.Service, logObj.Level, logObj.Message)))
	if err != nil {
		log.Warnf("Failed to send message to Kafka: %v", err)
	}
	s.counters.LogsReceived.Inc(logObj.Service, logObj.Level)
	id, err := s.logRepo.SendLog(ctx, logObj)
	if err != nil {
		return 0, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return id, nil
}

func (s *LogService) GetStats(ctx context.Context, service string, from, to time.Time) (domain.ServiceStats, error) {
	stats, err := s.logRepo.GetStatsByService(ctx, service, from, to)
	if err != nil {
		return domain.ServiceStats{}, errorsUtils.WrapPathErr(ErrCannotCreateLog)
	}
	return stats, nil
}
