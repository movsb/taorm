package mimic

import (
	"database/sql"
	"database/sql/driver"
	"testing"
)

func TestMimic(t *testing.T) {
	SetRows([]string{"id", "name", "age"}, [][]driver.Value{
		[]driver.Value{int64(1), "tao.yang", 100},
		[]driver.Value{int64(2), "jianqiao.hu", 101},
		[]driver.Value{int64(3), "daniel.zhang", 102},
	})

	db, err := sql.Open("mimic", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type User struct {
		ID   int64
		Name string
		Age  int
	}

	var users []*User
	rows, err := db.Query("select * from users")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Age); err != nil {
			t.Fatal(err)
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, user := range users {
		t.Logf("%+v", user)
	}
}
