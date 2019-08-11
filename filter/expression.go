package filter

import (
	"strconv"
)

// _ExpressionOverrider overrides expression.
type _ExpressionOverrider struct {
	Query string
	Args  []interface{}
}

// Expression is a filter expression.
// It is expressed as `key op value`.
type Expression struct {
	Name     string      // key name
	Operator Token       // op
	Type     ValueType   // value type
	Value    interface{} // raw value or set value

	nameChanged bool // is name set by user? if set, use it instead of converting it to column name

	// It is overrode by the user mapper.
	overrider *_ExpressionOverrider
}

// SetName sets the name of the expression.
func (e *Expression) SetName(name string) {
	e.Name = name
	e.nameChanged = true
}

// SetNumber sets the number value of the expression.
// TODO use interface{} to accept all numeric types.
func (e *Expression) SetNumber(num int64) {
	e.Type = ValueTypeNumber
	e.Value = num
}

// SetString sets the string value of the expression.
func (e *Expression) SetString(str string) {
	e.Type = ValueTypeString
	e.Value = str
}

// SetBoolean sets the boolean value of the expression.
func (e *Expression) SetBoolean(b bool) {
	e.Type = ValueTypeBoolean
	e.Value = b
}

// convertTo converts value to the field-specific type.
// If no conversion can be made, an error is returned.
func (e *Expression) convertTo(toType ValueType) error {
	bad := func() error {
		return newBadConversionError(e.Name, e.Value, e.Type, toType)
	}
	switch toType {
	case ValueTypeBoolean:
		switch e.Type {
		case ValueTypeBoolean:
			return nil
		case ValueTypeRaw:
			switch e.Value.(string) {
			case "1", "true":
				e.SetBoolean(true)
				return nil
			case "0", "false":
				e.SetBoolean(false)
				return nil
			}
		}
	case ValueTypeNumber:
		switch e.Type {
		case ValueTypeNumber:
			return nil
		case ValueTypeRaw:
			n, err := strconv.ParseInt(e.Value.(string), 10, 64)
			if err != nil {
				return bad()
			}
			e.SetNumber(n)
			return nil
		}
	case ValueTypeString:
		switch e.Type {
		case ValueTypeString:
			return nil
		case ValueTypeRaw:
			// nothing to do
			return nil
		}
	}
	return bad()
}
