package taorm

import (
	"database/sql"
)

// DB wraps sql.DB.
type DB struct {
	rdb *sql.DB // raw db
	_SQLCommon
}

// NewDB news a DB.
func NewDB(db *sql.DB) *DB {
	t := &DB{
		rdb:        db,
		_SQLCommon: db,
	}
	return t
}

// TxCall calls callback within transaction.
// It automatically catches and rethrows exceptions.
func (db *DB) TxCall(callback func(tx *DB) error) error {
	rtx, err := db.rdb.Begin()
	if err != nil {
		return WrapError(err)
	}

	tx := &DB{
		rdb:        db.rdb,
		_SQLCommon: rtx,
	}

	var exception struct {
		caught bool        // user callback threw an exception
		what   interface{} // user thrown exception
	}

	catchCall := func() (err error) {
		called := false
		defer func() {
			exception.what = recover()
			exception.caught = !called
		}()
		err = callback(tx)
		called = true
		return
	}

	if err := catchCall(); err != nil {
		rtx.Rollback()
		return err // user error, not wrapped
	}

	if exception.caught {
		rtx.Rollback()
		panic(exception.what) // user exception, not wrapped
	}

	if err = rtx.Commit(); err != nil {
		rtx.Rollback()
		return WrapError(err)
	}

	return nil
}

func (db *DB) Model(model interface{}) *Stmt {
	stmt := &Stmt{
		db:         db,
		model:      model,
		tableNames: []string{},
		limit:      -1,
		offset:     -1,
	}

	info, err := getRegistered(model)
	if err != nil {
		panic(WrapError(err))
	}

	stmt.tableNames = append(stmt.tableNames, info.tableName)

	stmt.info = info

	return stmt
}

func (db *DB) From(table string) *Stmt {
	stmt := &Stmt{
		db:         db,
		tableNames: []string{table},
		limit:      -1,
		offset:     -1,
	}
	return stmt
}

func (db *DB) Raw(query string, args ...interface{}) _Finder {
	stmt := &Stmt{}
	stmt.raw.query = query
	stmt.raw.args = args
	return stmt
}
