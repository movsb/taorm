package mimic

import (
	"database/sql"
	"database/sql/driver"
	"io"
)

func init() {
	sql.Register("mimic", &Driver{})
}

// Driver implements driver.Driver.
type Driver struct {
}

var _ driver.Driver = &Driver{}

// Open ...
func (d *Driver) Open(name string) (driver.Conn, error) {
	return &Conn{}, nil
}

// Conn implements driver.Conn.
type Conn struct {
}

var _ driver.Conn = &Conn{}

// Prepare ...
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return &Stmt{}, nil
}

// Close ...
func (c *Conn) Close() error {
	return nil
}

// Begin ...
func (c *Conn) Begin() (driver.Tx, error) {
	panic("tx not supported")
}

// Stmt  implements driver.Stmt.
type Stmt struct {
}

var _ driver.Stmt = &Stmt{}

// Close ...
func (s *Stmt) Close() error {
	return nil
}

// NumInput ...
func (s *Stmt) NumInput() int {
	return -1
}

// Exec ...
func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return &Result{}, nil
}

// Query ...
func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return &Rows{}, nil
}

type Result struct {
}

var _ driver.Result = &Result{}

func (r *Result) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *Result) RowsAffected() (int64, error) {
	return 0, nil
}

type Rows struct {
	index int
}

var _ driver.Rows = &Rows{}

func (r *Rows) Columns() []string {
	return _columns
}

func (r *Rows) Close() error {
	return nil
}

func (r *Rows) Next(dest []driver.Value) error {
	if r.index >= len(_values) {
		return io.EOF
	}
	for i, n := 0, len(dest); i < n; i++ {
		dest[i] = _values[r.index][i]
	}
	r.index++
	return nil
}
