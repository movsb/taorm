package filter

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Mapper maps
type Mapper map[string]interface{}

type Fielder func(field string) reflect.Type

type _Filter struct {
	mapper  Mapper
	fielder Fielder
}

// Filter news an filter and calls its Filter method.
func Filter(fielder Fielder, filter string, mapper Mapper) (query string, args []interface{}, err error) {
	f := &_Filter{
		mapper:  mapper,
		fielder: fielder,
	}
	return f.Filter(filter)
}

// Filter the result of the GORM query according to some conditions.
//
// Mapper prototypes are:
//
//   (strval)   => (strval)
//   (strval)   => (key, strval)
//   (strval)   => (intval)
//   (strval)   => (key, intval)
//   (strval)   => (boolval)
//   (strval)   => (key, boolval)
//   (intval)   => (intval)
//   (intval)   => (key, intval)
//   ()         => (enum)
//   ()         => (key, enum)
//   (enumItem) => (enumItem, enum)
//   (enumItem) => (key, enumItem, enum)
// Override:
//   (operator, strval)   => (query, args)
//   (operator, strval)   => ()
//
func (i *_Filter) Filter(filter string) (string, []interface{}, error) {
	tokenizer := NewTokenizer(filter)
	parser := NewParser(tokenizer)

	ast, err := parser.Parse()
	if err != nil {
		return "", nil, err
	}

	var query string
	var args []interface{}

	for index, and := range ast.AndExprs {
		orQuery, orArgs, err := i.filterAndExpression(and)
		if err != nil {
			return "", nil, err
		}
		if index != 0 {
			query += " AND "
		}
		if orQuery != "" {
			query += "(" + orQuery + ")"
			args = append(args, orArgs...)
		}
	}

	return query, args, nil
}

func (i *_Filter) callMapper(expr *Expression) error {
	mapper, ok := i.mapper[expr.Name]
	if !ok {
		return nil
	}

	// just in case. Already converted? no matter who did this.
	if expr.Type != ValueTypeRaw {
		return nil
	}
	raw, ok := expr.Value.(string)
	// shouldn't happen
	if !ok {
		return nil
	}

	switch typed := mapper.(type) {
	// (strval) => (strval)
	case func(string) string:
		val := typed(raw)
		expr.SetString(val)
	// (strval) => (key, strval)
	case func(string) (string, string):
		key, val := typed(raw)
		expr.SetName(key)
		expr.SetString(val)
	// (strval) => (intval)
	case func(string) int64:
		val := typed(raw)
		expr.SetNumber(val)
	// (strval) => (key, intval)
	case func(string) (string, int64):
		key, val := typed(raw)
		expr.SetName(key)
		expr.SetNumber(val)
	// (strval) => (boolval)
	case func(string) bool:
		val := typed(raw)
		expr.SetBoolean(val)
	// (strval) => (key, boolval)
	case func(string) (string, bool):
		key, val := typed(raw)
		expr.SetName(key)
		expr.SetBoolean(val)
	// (intval) => (intval)
	case func(int64) int64:
		num, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		val := typed(num)
		expr.SetNumber(val)
	// (intval) => (key, intval)
	case func(int64) (string, int64):
		num, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		key, val := typed(num)
		expr.SetName(key)
		expr.SetNumber(val)
	// () => enum
	case map[string]int32:
		val, ok := typed[raw]
		if !ok {
			return newEnumNotFoundError(expr.Name, raw)
		}
		expr.SetNumber(int64(val))
	// () => enum
	case func() map[string]int32:
		val, ok := typed()[raw]
		if !ok {
			return newEnumNotFoundError(expr.Name, raw)
		}
		expr.SetNumber(int64(val))
	// () => (key, enum)
	case func() (string, map[string]int32):
		key, values := typed()
		val, ok := values[raw]
		if !ok {
			return newEnumNotFoundError(key, raw)
		}
		expr.SetName(key)
		expr.SetNumber(int64(val))
	// (enumItem) => (enumItem, enum)
	case func(string) (string, map[string]int32):
		enumItem, values := typed(raw)
		val, ok := values[enumItem]
		if !ok {
			return newEnumNotFoundError(expr.Name, enumItem)
		}
		expr.SetNumber(int64(val))
	// (enumItem) => (key, enumItem, enum)
	case func(string) (string, string, map[string]int32):
		key, enumItem, values := typed(raw)
		val, ok := values[enumItem]
		if !ok {
			return newEnumNotFoundError(key, enumItem)
		}
		expr.SetName(key)
		expr.SetNumber(int64(val))
	// (operator, strval) => (query, args)
	case func(TokenType, string) (string, []interface{}):
		query, args := typed(expr.Operator.TokenType, raw)
		expr.overrider = &_ExpressionOverrider{
			Query: query,
			Args:  args,
		}
	// (operator, strval) => ()
	case func(TokenType, string):
		typed(expr.Operator.TokenType, raw)
		return errSkipFilter
	default:
		return &UnknownMapperError{}
	}
	return nil
}

func (i *_Filter) filterAndExpression(andExpr *AndExpression) (query string, args []interface{}, err error) {
	where := bytes.NewBuffer(nil)

	for index, expr := range andExpr.OrExprs {
		condition := ""

		// calls to mapper to see if user want to do some customizing
		if err := i.callMapper(expr); err != nil {
			switch err {
			default:
				return "", nil, err
			case errSkipFilter:
				continue
			}
		}

		if expr.overrider == nil {
			fType := i.fielder(expr.Name)
			if fType == nil {
				return "", nil, fmt.Errorf("filter: unknown field: %s", expr.Name)
			}
			vType := ValueTypeRaw
			switch fType.Kind() {
			case reflect.Int, reflect.Uint, reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16,
				reflect.Int32, reflect.Uint32, reflect.Int64, reflect.Uint64, reflect.Float32, reflect.Float64:
				vType = ValueTypeNumber
			case reflect.String:
				vType = ValueTypeString
			case reflect.Bool:
				vType = ValueTypeBoolean
			default:
				return "", nil, fmt.Errorf("filter: invalid field type to filter")
			}

			columnName := expr.Name

			if err := expr.convertTo(vType); err != nil {
				return "", nil, err
			}

			switch expr.Operator.TokenType {
			case TokenTypeEqual:
				condition = fmt.Sprintf("%s = ?", columnName)
			case TokenTypeNotEqual:
				condition = fmt.Sprintf("%s <> ?", columnName)
			case TokenTypeInclude:
				condition = fmt.Sprintf("%s LIKE ?", columnName)
			case TokenTypeNotInclude:
				condition = fmt.Sprintf("%s NOT LIKE ?", columnName)
			case TokenTypeStartsWith:
				condition = fmt.Sprintf("%s LIKE ?", columnName)
			case TokenTypeEndsWith:
				condition = fmt.Sprintf("%s LIKE ?", columnName)
			case TokenTypeMatch, TokenTypeNotMatch:
				return "", nil, fmt.Errorf("not supported operator: %s", expr.Operator.TokenValue)
			case TokenTypeGreaterThan:
				condition = fmt.Sprintf("%s > ?", columnName)
			case TokenTypeLessThan:
				condition = fmt.Sprintf("%s < ?", columnName)
			case TokenTypeGreaterThanOrEqual:
				condition = fmt.Sprintf("%s >= ?", columnName)
			case TokenTypeLessThanOrEqual:
				condition = fmt.Sprintf("%s <= ?", columnName)
			default:
				return "", nil, fmt.Errorf("unknown operator: %s", expr.Operator.TokenValue)
			}

			where.WriteString(condition)

			switch expr.Operator.TokenType {
			// must be string
			case TokenTypeInclude, TokenTypeNotInclude, TokenTypeStartsWith, TokenTypeEndsWith:
				search := strings.Replace(fmt.Sprintf("%v", expr.Value), "%", "%%", -1)
				format := "%s"
				switch expr.Operator.TokenType {
				case TokenTypeInclude, TokenTypeNotInclude:
					format = "%%%s%%"
				case TokenTypeStartsWith:
					format = "%s%%"
				case TokenTypeEndsWith:
					format = "%%%s"
				}
				args = append(args, fmt.Sprintf(format, search))
			default:
				switch expr.Type {
				case ValueTypeBoolean:
					// by default, we assume that boolean is stored as tinyint(1).
					if expr.Value.(bool) {
						args = append(args, 1)
					} else {
						args = append(args, 0)
					}
				default:
					args = append(args, expr.Value)
				}
			}
		} else {
			where.WriteString(expr.overrider.Query)
			args = append(args, expr.overrider.Args...)
		}

		if index < len(andExpr.OrExprs)-1 {
			where.WriteString(" OR ")
		}
	}

	return where.String(), args, nil
}
