package logger

import (
	"fmt"
	"path"
	"runtime"

	"github.com/Egor213/LogiTrack/internal/domain"
	log "github.com/sirupsen/logrus"
)

func SetupLogger(level string) {
	loggerLevel, err := log.ParseLevel(level)
	log.SetReportCaller(true)

	log.SetFormatter(&log.JSONFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return "", fmt.Sprintf("%s:%d", path.Base(frame.File), frame.Line)
		},
		TimestampFormat: "2006-01-02 15:04:05",
	})

	if err != nil {
		log.Infof("Level setup default INFO, err: %v", err)
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(loggerLevel)
	}

}

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
