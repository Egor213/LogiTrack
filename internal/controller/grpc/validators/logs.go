package validators

import (
	"errors"

	"github.com/Egor213/LogiTrack/internal/domain"
)

const (
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
	ErrEmptyService    = errors.New("service must be specified")
)

func Validate(l *domain.LogEntry) error {
	switch l.Level {
	case LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}

	if l.Service == "" {
		return ErrEmptyService
	}

	return nil
}
