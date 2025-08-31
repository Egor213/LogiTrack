package repotypes

import (
	"time"
)

type LogFilterInput struct {
	Service string
	Level   string
	From    time.Time
	To      time.Time
}
