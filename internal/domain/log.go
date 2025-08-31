package domain

import "time"

type LogEntry struct {
	Service   string
	Level     string
	Message   string
	Timestamp time.Time
}
