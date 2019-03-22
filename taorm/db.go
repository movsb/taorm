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
// It automatically catches and returns exceptions.
func (db *DB) TxCall(callback func(tx *DB) interface{}) interface{} {
	rtx, err := db.rdb.Begin()
	if err != nil {
		return err
	}

	tx := &DB{
		rdb: db.rdb,
		cdb: rtx,
	}

	catchCall := func() (except interface{}) {
		defer func() {
			except = recover()
		}()
		return callback(tx)
	}

	if except := catchCall(); except != nil {
		rtx.Rollback()
		return except
	}

	if err := rtx.Commit(); err != nil {
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
