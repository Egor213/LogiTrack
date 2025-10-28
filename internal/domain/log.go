package domain

import "time"

type LogEntry struct {
	Id        string    `db:"id"`
	Service   string    `db:"service"`
	Level     string    `db:"level"`
	Message   string    `db:"message"`
	Timestamp time.Time `db:"created_at"`
}

type LevelStats struct {
	Level string
	Count int
}

type ServiceStats struct {
	Service     string
	TotalLogs   int
	LogsByLevel []LevelStats
}
