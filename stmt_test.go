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

func (User) TableName() string {
	return `users`
}

type Like struct {
	ID     int64
	UserID int64
	LikeID int64
}

func (Like) TableName() string {
	return `likes`
}

func TestSQLs(t *testing.T) {
	db, err := sql.Open("mysql", "taorm:taorm@/taorm")
	if err != nil {
		t.Fatal(err)
	}
	tdb := NewDB(db)
	tests := []struct {
		want string
		got  string
	}{
		{
			"SELECT * FROM users",
			tdb.From(User{}).FindSQL(),
		},
		{
			"SELECT * FROM users",
			tdb.From(User{ID: 28}).FindSQL(),
		},
		{
			"INSERT INTO users (name,age) VALUES ('tao',18)",
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
			"UPDATE users SET name='TAO',age=28 WHERE (id=1)",
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
			"UPDATE users SET age=age+1",
			tdb.Model(User{
				Age: 18,
			}).UpdateMapSQL(M{
				"age": Expr("age+?", 1),
			}),
		},
		{
			"DELETE FROM users WHERE (id=1)",
			tdb.From(User{}).Where("id=?", 1).DeleteSQL(),
		},
		{
			"DELETE FROM users WHERE (id=28)",
			tdb.Model(User{ID: 28}).DeleteSQL(),
		},
		{
			"SELECT COUNT(1) FROM users WHERE (age=28)",
			tdb.From(User{}).Where("age=?", 28).CountSQL(),
		},
		{
			"SELECT * FROM users WHERE (id=1 AND age=28)",
			tdb.Raw("SELECT * FROM users WHERE (id=? AND age=?)", 1, 28).FindSQL(),
		},
		{
			"SELECT * FROM users ORDER BY id",
			tdb.From(User{}).OrderBy("id").FindSQL(),
		},
		{
			"SELECT users.* FROM users INNER JOIN likes ON users.id = likes.user_id ORDER BY id",
			tdb.From(User{}).OrderBy("id").InnerJoin(Like{}, "users.id = likes.user_id").FindSQL(),
		},
		{
			"SELECT users.* FROM users INNER JOIN likes ON users.id = likes.user_id ORDER BY users.id",
			tdb.From(User{}).OrderBy("users.id").InnerJoin(Like{}, "users.id = likes.user_id").FindSQL(),
		},
		{
			`SELECT * FROM users WHERE (name IN (?))`,
			tdb.From(User{}).Where(`name IN (?)`, []string{`tao`}).FindSQLRaw(),
		},
		{
			`SELECT * FROM users WHERE (name IN (?,?))`,
			tdb.From(User{}).Where(`name IN (?)`, []string{`tao`, `yang`}).FindSQLRaw(),
		},
		{
			`SELECT * FROM users WHERE (name IN ('tao','yang'))`,
			tdb.From(User{}).Where(`name IN (?)`, []string{`tao`, `yang`}).FindSQL(),
		},
	}
	for _, test := range tests {
		if test.want != test.got {
			t.Fatalf("not equal: \n    want: %s\n     got: %s\n", test.want, test.got)
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
		{int64(1), "tao", 100},
		{int64(2), "qiao", 101},
		{int64(3), "daniel", 102},
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
		{int64(1), "tao", 100},
		{int64(2), "qiao", 101},
		{int64(3), "daniel", 102},
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
