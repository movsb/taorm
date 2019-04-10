package taorm

import (
	"database/sql"
)

// DB wraps sql.DB.
type DB struct {
	rdb *sql.DB    // raw db
	cdb _SQLCommon // common db
}

// NewDB news a DB.
func NewDB(db *sql.DB) *DB {
	t := &DB{
		rdb: db,
		cdb: db,
	}
	return t
}

// TxCall calls callback within transaction.
// It automatically catches and rethrows exceptions.
func (db *DB) TxCall(callback func(tx *DB) error) error {
	rtx, err := db.rdb.Begin()
	if err != nil {
		return err
	}

	tx := &DB{
		rdb: db.rdb,
		cdb: rtx,
	}

	var exception struct {
		caught bool
		what   interface{}
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
		return err
	}

	if exception.caught {
		rtx.Rollback()
		panic(exception.what)
	}

	if err = rtx.Commit(); err != nil {
		rtx.Rollback()
		return err
	}

	return nil
}

func (db *DB) Model(model interface{}, name string) *Stmt {
	stmt := &Stmt{
		db:         db,
		model:      model,
		tableNames: []string{name},
		limit:      -1,
		offset:     -1,
	}

	stmt.initPrimaryKey()

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
