package taorm

import (
	"database/sql"
)

type _SQLCommon interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

// M is a string-interface map that is used for Update*.
type M map[string]any

// Finder wraps method for SELECT.
type Finder interface {
	Find(out any) error
	MustFind(out any)
	FindSQL() string
	Count(out any) error
	MustCount(out any)
	CountSQL() string
}

// TableNamer ...
type TableNamer interface {
	TableName() string
}
