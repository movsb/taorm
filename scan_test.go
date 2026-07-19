package taorm

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/movsb/taorm/mimic"
)

func TestScan(t *testing.T) {
	mimic.SetRows([]string{"id", "name", "age"}, [][]driver.Value{
		{int64(1), "tao", 100},
		{int64(2), "xxx", 101},
	})

	db, err := sql.Open("mimic", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var users []User
	MustScanRows(&users, db, `select 1`)
}
