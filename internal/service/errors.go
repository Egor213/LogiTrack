package service

import "fmt"

var (
	ErrLogAlreadyExists = fmt.Errorf("log already exists")
	ErrCannotCreateLog  = fmt.Errorf("cannot create log")
)
