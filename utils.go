package taorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unsafe"
)

// https://gist.github.com/stoewer/fbe273b711e6a06315d19552dd4d33e60
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

var scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
var valuerType = reflect.TypeOf((*driver.Valuer)(nil)).Elem()

func isColumnField(field reflect.StructField) bool {
	if !ast.IsExported(field.Name) {
		return false
	}

	t := field.Type

	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}

	valueable := t.Implements(valuerType)
	scannable := reflect.PtrTo(t).Implements(scannerType) && !t.Implements(scannerType)
	if valueable && scannable {
		return true
	}

	dummy := reflect.NewAt(t, unsafe.Pointer(nil)).Interface()
	switch dummy.(type) {
	case *time.Time:
		return true
	case *[]byte:
		return true
	}

	return false
}

func getColumnName(field reflect.StructField) string {
	tag := field.Tag.Get("taorm")
	kvs := strings.Split(tag, ",")
	for _, kv := range kvs {
		s := strings.Split(kv, ":")
		switch s[0] {
		case "-":
			return ""
		case "name":
			if len(s) > 1 {
				return s[1]
			}
			return ""
		}
	}
	return toSnakeCase(field.Name)
}

type _EmptyEface struct {
	typ *struct{}
	ptr unsafe.Pointer
}

// createSQLInMarks creates "?,?,?" string.
func createSQLInMarks(count int) string {
	s := "?"
	for i := 1; i < count; i++ {
		s += ",?"
	}
	return s
}

func panicIf(cond bool, v interface{}) {
	if cond {
		panic(v)
	}
}

func dumpSQL(query string, args ...interface{}) {
	// fmt.Println(strSQL(query, args...))
}

type _StrArg struct {
	a any
}

func (v _StrArg) String() string {
	switch typed := v.a.(type) {
	case string:
		return fmt.Sprintf(`'%s'`, strings.ReplaceAll(typed, `'`, `\'`))
	default:
		return fmt.Sprint(typed)
	}
}

func strSQL(query string, args ...interface{}) string {
	sa := make([]_StrArg, 0, len(args))
	for _, a := range args {
		sa = append(sa, _StrArg{a: a})
	}
	for i := range args {
		args[i] = sa[i]
	}
	return fmt.Sprintf(strings.ReplaceAll(query, "?", "%v"), args...)
}

func structName(ty reflect.Type) string {
	return ty.PkgPath() + "." + ty.Name()
}
