package taorm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-sql-driver/mysql"
)

func WrapError(err error) error {
	if myErr, ok := err.(*mysql.MySQLError); ok {
		switch myErr.Number {
		case 1062:
			return &DupKeyError{}
		}
	}
	return err
}

var (
	ErrNoWhere    = errors.New("taorm: no wheres")
	ErrNoFields   = errors.New("taorm: no fields")
	ErrInvalidOut = errors.New("taorm: invalid out")
	ErrDupKey     = errors.New("taorm: dup key")
)

type DupKeyError struct {
}

func (e DupKeyError) Error() string {
	return "dup key error"
}

// NoPlaceToSaveFieldError ...
type NoPlaceToSaveFieldError struct {
	Field string
}

func (e NoPlaceToSaveFieldError) Error() string {
	return fmt.Sprintf("taorm: no place to save field `%s'", e.Field)
}

// UnknownFieldKindError ...
type UnknownFieldKindError struct {
	Field string
	Kind  reflect.Kind
}

func (e UnknownFieldKindError) Error() string {
	return fmt.Sprintf("taorm: unknown field kind (field: `%s', kind: `%v')", e.Field, e.Kind)
}

// NotStructError ...
type NotStructError struct {
	Kind reflect.Kind
}

func (e NotStructError) Error() string {
	return fmt.Sprintf("taorm: not a struct: `%v'", e.Kind)
}
