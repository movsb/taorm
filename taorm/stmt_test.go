package taorm

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/movsb/taorm/mimic"
)

type User struct {
	ID   int64
	Name string
	Age  int
}

func TestSQLs(t *testing.T) {
	db, err := sql.Open("mysql", "taorm:taorm@/taorm")
	if err != nil {
		t.Fatal(err)
	}
	tdb := NewDB(db)
	Register(User{}, "users")
	tests := []struct {
		want string
		got  string
	}{
		{
			"SELECT * FROM users",
			tdb.From("users").FindSQL(),
		}, {
			"INSERT INTO users (name,age) VALUES (tao,18)",
			tdb.Model(User{
				Name: "tao",
				Age:  18,
			}).CreateSQL(),
		},
		{
			"UPDATE users SET age=20 WHERE (id=1)",
			tdb.Model(User{
				ID:   1,
				Name: "tao",
				Age:  18,
			}).UpdateMapSQL(M{
				"age": 20,
			}),
		},
		{
			"UPDATE users SET name=TAO,age=28 WHERE (id=1)",
			tdb.Model(User{
				ID:   1,
				Name: "tao",
				Age:  18,
			}).UpdateModelSQL(User{
				ID:   1,
				Name: "TAO",
				Age:  28,
			}),
		},
		{
			"DELETE FROM users WHERE (id=1)",
			tdb.From("users").Where("id=?", 1).DeleteSQL(),
		},
	}
	for _, test := range tests {
		if test.want != test.got {
			t.Fatal("not equal: ", test.want, "!=", test.got)
		}
	}
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
		err := taodb.Model(&user).Create()
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
