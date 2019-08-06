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

// MustTxCall ...
func (db *DB) MustTxCall(callback func(tx *DB)) {
	if err := db.TxCall(func(tx *DB) error {
		callback(tx)
		return nil
	}); err != nil {
		panic(err)
	}
}

// Model ...
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

// From ...
func (db *DB) From(table interface{}) *Stmt {
	stmt := &Stmt{
		db:     db,
		limit:  -1,
		offset: -1,
	}
	if err := stmt.tryFindTableName(table); err != nil {
		panic(WrapError(err))
	}
	return stmt
}

// Raw executes a raw SQL query that returns rows.
func (db *DB) Raw(query string, args ...interface{}) Finder {
	stmt := &Stmt{
		db: db,
	}
	stmt.raw.query = query
	stmt.raw.args = args
	return stmt
}

// --- stmt impl. ---
//
// Below are some commonly used functions to begin a preparing.

// MustExec ...
func (db *DB) MustExec(query string, args ...interface{}) sql.Result {
	result, err := db.Exec(query, args...)
	if err != nil {
		panic(WrapError(err))
	}
	return result
}

func (db *DB) _New() *Stmt {
	stmt := &Stmt{
		db:     db,
		limit:  -1,
		offset: -1,
	}
	return stmt
}

// Select ...
func (db *DB) Select(fields string) *Stmt {
	return db._New().Select(fields)
}

// Where ...
func (db *DB) Where(query string, args ...interface{}) *Stmt {
	return db._New().Where(query, args...)
}

// WhereIf ...
func (db *DB) WhereIf(cond bool, query string, args ...interface{}) *Stmt {
	return db._New().WhereIf(cond, query, args...)
}

// Find ...
func (db *DB) Find(out interface{}) error {
	return db._New().Find(out)
}

// MustFind ...
func (db *DB) MustFind(out interface{}) {
	db._New().MustFind(out)
}
