package taorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-sql-driver/mysql"
)

var (
	reErr1062 = regexp.MustCompile(`Duplicate entry '([^']*)' for key '([^']+)'`)
)

// Error all errors wrapper.
type Error struct {
	Err error
	Raw error
}

func (e Error) Error() string {
	return fmt.Sprintf("taorm error: %v: %v", e.Err, e.Raw)
}

// WrapError wraps all errors to tao error.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	// don't wrap again
	if _, ok := err.(*Error); ok {
		return err
	}

	if myErr, ok := err.(*mysql.MySQLError); ok {
		switch myErr.Number {
		case 1062:
			matches := reErr1062.FindStringSubmatch(myErr.Message)
			return &Error{
				Err: &DupKeyError{
					Key:   matches[2],
					Value: matches[1],
				},
				Raw: myErr,
			}
		}
	}

	switch err {
	case sql.ErrNoRows:
		return &Error{Err: &NotFoundError{}, Raw: err}
	}

	switch err {
	case ErrInternal:
	case ErrNoWhere:
	case ErrNoFields:
	case ErrInvalidOut:
		return &Error{Err: ErrInternal, Raw: err}
	}

	switch err.(type) {
	case *NoPlaceToSaveFieldError:
		return &Error{Err: ErrInternal, Raw: err}
	case *NotStructError:
		return &Error{Err: ErrInternal, Raw: err}
	}

	return &Error{Err: ErrInternal, Raw: err}
}

var (
	ErrInternal   = errors.New("taorm: internal error")
	ErrNoWhere    = errors.New("taorm: no wheres")
	ErrNoFields   = errors.New("taorm: no fields")
	ErrInvalidOut = errors.New("taorm: invalid out")
)

type NotFoundError struct {
}

func (e NotFoundError) Error() string {
	return "Not Found"
}

// DupKeyError ...
type DupKeyError struct {
	Key   string
	Value string
}

func (e DupKeyError) Error() string {
	return fmt.Sprintf("DupKeyError: key=%s,value=%s", e.Key, e.Value)
}

// NoPlaceToSaveFieldError ...
type NoPlaceToSaveFieldError struct {
	Field string
}

func (e NoPlaceToSaveFieldError) Error() string {
	return fmt.Sprintf("NoPlaceToSaveFieldError: `%s'", e.Field)
}

// NotStructError ...
type NotStructError struct {
	Kind reflect.Kind
}

func (e NotStructError) Error() string {
	return fmt.Sprintf("taorm: not a struct: `%v'", e.Kind)
}

func IsNotFoundError(err error) bool {
	if err == sql.ErrNoRows {
		return true
	}
	if te, ok := err.(*Error); ok {
		if te.Raw == sql.ErrNoRows {
			return true
		}
	}
	return false
}
