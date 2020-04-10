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
