package logginghelper

import (
	"github.com/Egor213/LogiTrack/internal/domain"
	log "github.com/sirupsen/logrus"
)

func LogReceived(entry *domain.LogEntry) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"message": entry.Message,
	}).Info("Received log via gRPC")
}

func LogSaved(entry *domain.LogEntry, id int) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"id":      id,
	}).Info("Log saved successfully")
}

func LogError(entry *domain.LogEntry, err error) {
	log.WithFields(log.Fields{
		"service": entry.Service,
		"level":   entry.Level,
		"error":   err,
	}).Error("Failed to save log")
}
