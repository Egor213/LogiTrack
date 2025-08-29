package errorsUtils

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
	CodeNotNullViolation    = "23502"
)

func Is(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}

func IsUniqueViolation(err error) bool {
	return Is(err, CodeUniqueViolation)
}

func IsForeignKeyViolation(err error) bool {
	return Is(err, CodeForeignKeyViolation)
}

func IsNotNullViolation(err error) bool {
	return Is(err, CodeNotNullViolation)
}

func WrapPathErr(err error) error {
	pc, _, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc).Name()
	return fmt.Errorf("[%s:%d] %w", fn, line, err)
}
