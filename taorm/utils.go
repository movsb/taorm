package taorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strings"
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

func isColumnField(field reflect.StructField) bool {
	if !ast.IsExported(field.Name) {
		return false
	}
	if field.Type.Kind() == reflect.Struct || field.Type.Kind() == reflect.Ptr {
		dummy := reflect.NewAt(field.Type, unsafe.Pointer(nil)).Interface()
		_, scanable := dummy.(sql.Scanner)
		_, valueable := dummy.(driver.Valuer)
		return scanable && valueable
	}
	return true
}

func getColumnName(field reflect.StructField) string {
	tag := field.Tag.Get("taorm")
	kvs := strings.Split(tag, ",")
	for _, kv := range kvs {
		s := strings.Split(kv, ":")
		switch s[0] {
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

func strSQL(query string, args ...interface{}) string {
	return fmt.Sprintf(strings.Replace(query, "?", "%v", -1), args...)
}

func structName(ty reflect.Type) string {
	return ty.PkgPath() + "." + ty.Name()
}
