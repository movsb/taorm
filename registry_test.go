package taorm

import (
	"reflect"
	"testing"
)

type TableNameType1 struct {
}

func (TableNameType1) TableName() string {
	return `table_name`
}

type TableNameType2 struct {
}

func (*TableNameType2) TableName() string {
	return `table_name`
}

func assertEqual(t *testing.T, a, b any) {
	if a != b {
		t.Fatal(`not equal`)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTableName(t *testing.T) {
	{
		name, err := getTableNameFromType(reflect.TypeOf(TableNameType1{}))
		assertNoError(t, err)
		assertEqual(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(&TableNameType1{}))
		assertNoError(t, err)
		assertEqual(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(TableNameType2{}))
		assertNoError(t, err)
		assertEqual(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(&TableNameType2{}))
		assertNoError(t, err)
		assertEqual(t, `table_name`, name)
	}
}

type Base struct {
	ID int64 `taorm:"id"`
}

type lowerBase struct {
	Lower int64 `taorm:"lower"`
}

type Embedded struct {
	Base
	lowerBase
	Name string `taorm:"name:name"`
}

func TestEmbedded(t *testing.T) {
	info, err := getRegistered(Embedded{})
	if err != nil {
		t.Fatal(err)
	}

	names := []string{`id`, `name`, `lower`}
	for _, name := range names {
		if _, ok := info.fields[name]; !ok {
			t.Fatalf(`name not in fields: %s`, name)
		}
	}
	if len(info.fields) != len(names) {
		t.Fatalf(`len(names) not equal: %d vs %d`, len(info.fields), len(names))
	}
}
