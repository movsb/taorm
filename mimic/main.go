// Package mimic is a mock/mimic/dummy sql driver that
// supports in-memory exec/query for sql benchmarks.
package mimic

import (
	"database/sql/driver"
)

var _columns []string
var _values [][]driver.Value

func SetRows(columns []string, values [][]driver.Value) {
	_columns = columns
	_values = values
}
