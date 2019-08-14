package filter

import (
	"errors"
	"fmt"
)

var errSkipFilter = errors.New("filter: should skip")

// UnknownMapperError implies an unknown mapper error.
type UnknownMapperError struct {
}

func (e UnknownMapperError) Error() string {
	return "insight: unknown mapper prototype"
}

// InvalidOrderByError implies that an order by statement is invalid.
type InvalidOrderByError struct {
}

func (e InvalidOrderByError) Error() string {
	return "insight: invalid order by"
}

// StructNotRegisteredError implies that a struct type has not been registered.
type StructNotRegisteredError struct {
}

func (e StructNotRegisteredError) Error() string {
	return "insight: struct is not registered"
}

// EnumNotFoundError implies an enum was not found.
type EnumNotFoundError struct {
	enum string
	key  string
}

// NewEnumNotFoundError news a enum not found error.
func newEnumNotFoundError(enum string, key string) *EnumNotFoundError {
	return &EnumNotFoundError{
		enum: enum,
		key:  key,
	}
}

func (e EnumNotFoundError) Error() string {
	return fmt.Sprintf(`insight: enum not found: %s["%s"]`, e.enum, e.key)
}

// BadConversionError implies a conversion cannot be made.
type BadConversionError struct {
	field string
	value interface{}
	from  ValueType
	to    ValueType
}

func newBadConversionError(field string, value interface{}, from ValueType, to ValueType) *BadConversionError {
	return &BadConversionError{
		field: field,
		value: value,
		from:  from,
		to:    to,
	}
}

func (e BadConversionError) Error() string {
	return fmt.Sprintf(`insight: bad conversion: cannot convert value %v from type %s to type %s for field %s`, e.value, e.from, e.to, e.field)
}

// SyntaxError implies a syntax error.
type SyntaxError struct {
	msg string
}

// NewSyntaxError news a syntax error.
func newSyntaxError(format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{
		msg: fmt.Sprintf(format, args...),
	}
}

func (e SyntaxError) Error() string {
	return "insight: " + e.msg
}
