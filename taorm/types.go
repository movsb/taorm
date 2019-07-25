package taorm

import "database/sql"

type _SQLCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// Expr is raw SQL string.
type Expr string

type M map[string]interface{}

type _Finder interface {
	Find(out interface{}) error
	MustFind(out interface{})
	FindSQL() string
	Count(out interface{}) error
	MustCount(out interface{})
	CountSQL() string
}
