package taorm

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-sql-driver/mysql"
)

var (
	reErr1062 = regexp.MustCompile(`Duplicate entry '([^']+)' for key '([^']+)'`)
)

func WrapError(err error) error {
	if myErr, ok := err.(*mysql.MySQLError); ok {
		switch myErr.Number {
		case 1062:
			matches := reErr1062.FindStringSubmatch(myErr.Message)
			return &DupKeyError{
				Key:   matches[2],
				Value: matches[1],
			}
		}
	}
	return err
}

var (
	ErrNoWhere    = errors.New("taorm: no wheres")
	ErrNoFields   = errors.New("taorm: no fields")
	ErrInvalidOut = errors.New("taorm: invalid out")
)

type DupKeyError struct {
	Key   string
	Value string
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
	Type  reflect.Type
}

func (e UnknownFieldKindError) Error() string {
	return fmt.Sprintf("taorm: unknown field kind (field: `%s', kind: `%v')", e.Field, e.Type.String())
}

// NotStructError ...
type NotStructError struct {
	Kind reflect.Kind
}

func (e NotStructError) Error() string {
	return fmt.Sprintf("taorm: not a struct: `%v'", e.Kind)
}
