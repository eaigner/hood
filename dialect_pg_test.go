package hood

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"testing"
)

func TestPgInsert(t *testing.T) {
	db, err := sql.Open("postgres", "user=hood sslmode=disable")
	if err != nil {
		t.Fatal("could not open db", err)
	}
	dbName := "hood_test"
	hood := New(db, &DialectPg{})
	hood.Log = true
	err = hood.CreateDatabase(dbName)
	if err != nil {
		t.Fatal("could not create db", err)
	}
	defer hood.DropDatabase(dbName)
}
