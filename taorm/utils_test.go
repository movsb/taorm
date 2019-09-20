package taorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

type ColumnFieldStruct struct {
	unexported int            "false"
	B          bool           "true"
	I          int            "true"
	I8         int8           "true"
	I16        int16          "true"
	I32        int32          "true"
	I64        int64          "true"
	U          uint           "true"
	U8         uint8          "true"
	U16        uint16         "true"
	U32        uint32         "true"
	U64        uint64         "true"
	UPtr       uintptr        "false"
	F32        float32        "true"
	F64        float64        "true"
	C64        complex64      "false"
	C128       complex128     "false"
	A          [1]int         "false"
	C          chan string    "false"
	F          func()         "false"
	If         interface{}    "false"
	M          map[string]int "false"
	P          *int           "false"
	Slice      []int          "false"
	S          string         "true"
	Struct     struct{}       "false"
	UP         unsafe.Pointer "false"

	NullBool    sql.NullBool    "true"
	NullFloat64 sql.NullFloat64 "true"
	NullInt64   sql.NullInt64   "true"
	NullString  sql.NullString  "true"

	Time  time.Time "true"
	Bytes []byte    "true"

	TypeWithScannerAndValuer      _TypeWithScannerAndValuer      "true"
	TypeWithValueScannerAndValuer _TypeWithValueScannerAndValuer "false"
}

type _TypeWithScannerAndValuer struct{}

func (t _TypeWithScannerAndValuer) Value() (driver.Value, error) {
	return "", nil
}

func (t *_TypeWithScannerAndValuer) Scan(value interface{}) error {
	return nil
}

type _TypeWithValueScannerAndValuer struct{}

func (t _TypeWithValueScannerAndValuer) Value() (driver.Value, error) {
	return "", nil
}

func (t _TypeWithValueScannerAndValuer) Scan(value interface{}) error {
	return nil
}

func TestIsColumnField(t *testing.T) {
	typ := reflect.TypeOf(ColumnFieldStruct{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		b := isColumnField(f)
		want := string(f.Tag)
		got := fmt.Sprint(b)
		if got != want {
			t.Errorf("%-16s want=%-8s got=%-8s\n", f.Name, want, got)
		}
	}
}
