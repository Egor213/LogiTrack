package repotypes

import (
	"time"
)

// кажется, что это все таки бизнес сущность
type LogFilter struct {
	Service string
	Level   string
	From    time.Time
	To      time.Time
	Limit   int
}
