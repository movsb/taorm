package taorm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestGetTableName(t *testing.T) {
	{
		name, err := getTableNameFromType(reflect.TypeOf(TableNameType1{}))
		assert.NoError(t, err)
		assert.Equal(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(&TableNameType1{}))
		assert.NoError(t, err)
		assert.Equal(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(TableNameType2{}))
		assert.NoError(t, err)
		assert.Equal(t, `table_name`, name)
	}
	{
		name, err := getTableNameFromType(reflect.TypeOf(&TableNameType2{}))
		assert.NoError(t, err)
		assert.Equal(t, `table_name`, name)
	}
}

type Base struct {
	ID int64 `taorm:"id"`
}

type Embedded struct {
	Base
	Name string `taorm:"name:name"`
}

func TestEmbedded(t *testing.T) {
	info, err := getRegistered(Embedded{})
	if err != nil {
		t.Fatal(err)
	}

	names := []string{`id`, `name`}
	for _, name := range names {
		if _, ok := info.fields[name]; !ok {
			t.Fatalf(`name not in fields: %s`, name)
		}
	}
	if len(info.fields) != len(names) {
		t.Fatalf(`len(names) not equal: %d vs %d`, len(info.fields), len(names))
	}
}
