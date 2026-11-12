package repository

import (
	"context"
	"errors"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type ErrorClassification int

const (
	maxAttempts = 3
)

const (
	NonRetriable ErrorClassification = iota
	Retriable
)

var delayTimes = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

func Classify(err error) ErrorClassification {
	if err == nil {
		return NonRetriable
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return ClassifyPgError(pgErr)
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return Retriable
	}

	return NonRetriable
}

func ClassifyPgError(pgErr *pgconn.PgError) ErrorClassification {
	switch pgErr.Code {
	// Класс 08 - Ошибки соединения
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return Retriable

	// Класс 40 - Откат транзакции
	case pgerrcode.TransactionRollback, // 40000
		pgerrcode.SerializationFailure, // 40001
		pgerrcode.DeadlockDetected:     // 40P01
		return Retriable

	// Класс 57 - Ошибка оператора
	case pgerrcode.CannotConnectNow: // 57P03
		return NonRetriable
	}

	switch pgErr.Code {
	// Класс 22 - Ошибки данных
	case pgerrcode.DataException,
		pgerrcode.NullValueNotAllowedDataException:
		return NonRetriable

	// Класс 23 - Нарушение ограничений целостности
	case pgerrcode.IntegrityConstraintViolation,
		pgerrcode.RestrictViolation,
		pgerrcode.NotNullViolation,
		pgerrcode.ForeignKeyViolation,
		pgerrcode.UniqueViolation,
		pgerrcode.CheckViolation:
		return NonRetriable

	// Класс 42 - Синтаксические ошибки
	case pgerrcode.SyntaxErrorOrAccessRuleViolation,
		pgerrcode.SyntaxError,
		pgerrcode.UndefinedColumn,
		pgerrcode.UndefinedTable,
		pgerrcode.UndefinedFunction:
		return NonRetriable
	}

	return NonRetriable
}

func Retry(ctx context.Context, operation func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts+1; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err
		if Classify(err) != Retriable {
			return err
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(
			time.Duration(delayTimes[attempt]),
		):
		}
	}
	return lastErr
}
