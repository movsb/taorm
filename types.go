package taorm

import (
	"database/sql"
)

type _SQLCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// M is a string-interface map that is used for Update*.
type M map[string]interface{}

// Finder wraps method for SELECT.
type Finder interface {
	Find(out interface{}) error
	MustFind(out interface{})
	FindSQL() string
	Count(out interface{}) error
	MustCount(out interface{})
	CountSQL() string
}

// TableNamer ...
type TableNamer interface {
	TableName() string
}
