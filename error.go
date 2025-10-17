package taorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
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
//
// If err is nil, wrapped error is nil too.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	// don't wrap again
	if _, ok := err.(*Error); ok {
		return err
	}

	// official error constants
	switch err {
	case sql.ErrNoRows:
		return &Error{Err: &NotFoundError{}, Raw: err}
	}

	// taorm error constants
	switch err {
	case ErrInternal, ErrNoWhere, ErrNoFields, ErrInvalidOut:
		return &Error{Err: ErrInternal, Raw: err}
	}

	// taorm detailed errors
	switch err.(type) {
	case *NoPlaceToSaveFieldError:
		return &Error{Err: ErrInternal, Raw: err}
	case *NotStructError:
		return &Error{Err: ErrInternal, Raw: err}
	}

	// unhandled errors
	return &Error{Err: ErrInternal, Raw: err}
}

var (
	// ErrInternal ...
	ErrInternal = errors.New("internal error")
	// ErrNoWhere ...
	ErrNoWhere = errors.New("no wheres")
	// ErrNoFields ...
	ErrNoFields = errors.New("no fields")
	// ErrInvalidOut ...
	ErrInvalidOut = errors.New("invalid out")
)

// NotFoundError ...
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

// IsNotFoundError ...
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
