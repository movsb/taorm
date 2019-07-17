package taorm

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/movsb/taorm/mimic"
)

type User struct {
	ID   int64
	Name string
	Age  int
}

func BenchmarkInsert(b *testing.B) {
	user := User{
		Name: "tao",
		Age:  18,
	}

	db, err := sql.Open("mimic", "")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	taodb := NewDB(db)

	for i := 0; i < b.N; i++ {
		err := taodb.Model(&user, "users").Create()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectAll(b *testing.B) {
	mimic.SetRows([]string{"id", "name", "age"}, [][]driver.Value{
		[]driver.Value{int64(1), "tao", 100},
		[]driver.Value{int64(2), "qiao", 101},
		[]driver.Value{int64(3), "daniel", 102},
	})

	db, err := sql.Open("mimic", "")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	taodb := NewDB(db)

	for i := 0; i < b.N; i++ {
		var users []*User
		err := taodb.From("users").Find(&users)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectComplex(b *testing.B) {
	mimic.SetRows([]string{"id", "name", "age"}, [][]driver.Value{
		[]driver.Value{int64(1), "tao", 100},
		[]driver.Value{int64(2), "qiao", 101},
		[]driver.Value{int64(3), "daniel", 102},
	})

	db, err := sql.Open("mimic", "")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	taodb := NewDB(db)

	type User struct {
		ID   int64
		Name string
		Age  int
	}

	for i := 0; i < b.N; i++ {
		var users []*User
		err = taodb.From("users").Where("id > ?", 1).
			Where("name <> ?", "thing").
			Limit(1).
			GroupBy("id").
			Offset(1).
			Select("id, name, color, uuid, identifier, cargo, manifest").
			Find(&users)
		if err != nil {
			b.Fatal(err)
		}
	}
}
