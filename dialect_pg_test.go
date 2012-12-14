package hood

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"reflect"
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

func TestSqlType(t *testing.T) {
	d := &DialectPg{}
	if x := d.SqlType(reflect.TypeOf(true), 0, false); x != "boolean" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf(uint32(2)), 0, false); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf(int(1)), 0, true); x != "serial" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf(int64(1)), 0, false); x != "bigint" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf(int64(1)), 0, true); x != "bigserial" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf(1.8), 0, true); x != "double precision" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf([]byte("asdf")), 0, true); x != "bytea" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf("astring"), 0, true); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf([]bool{}), 0, true); x != "varchar(255)" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(reflect.TypeOf([]bool{}), 128, true); x != "varchar(128)" {
		t.Fatal("wrong type", x)
	}
}
