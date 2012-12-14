package hood

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"testing"
)

type PgDialectModel struct {
	Prim   int    `pk:"true"auto:"true"`
	First  string `null:"true"`
	Last   string `default:"'defaultValue'"`
	Amount int
}

func setupDb(t *testing.T) (*Hood, string) {
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

	return hood, dbName
}

func TestPgInsert(t *testing.T) {
	hood, dbName := setupDb(t)
	defer hood.DropDatabase(dbName)

	// TODO: complete
}

func TestSqlType(t *testing.T) {
	d := &DialectPg{}
	if x := d.SqlType(true, 0, false); x != "boolean" {
		t.Fatal("wrong type", x)
	}
	var indirect interface{} = true
	if x := d.SqlType(indirect, 0, false); x != "boolean" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(uint32(2), 0, false); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int(1), 0, true); x != "serial" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int64(1), 0, false); x != "bigint" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int64(1), 0, true); x != "bigserial" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(1.8, 0, true); x != "double precision" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]byte("asdf"), 0, true); x != "bytea" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType("astring", 0, true); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]bool{}, 0, true); x != "varchar(255)" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]bool{}, 128, true); x != "varchar(128)" {
		t.Fatal("wrong type", x)
	}
}

func TestCreateTableSql(t *testing.T) {
	hood, dbName := setupDb(t)
	defer hood.DropDatabase(dbName)

	model, _ := interfaceToModel(&PgDialectModel{})
	query := hood.createTableSql(model)
	if query != `CREATE TABLE pg_dialect_model ( prim serial PRIMARY KEY, first text, last text DEFAULT 'defaultValue', amount integer )` {
		t.Fatal("wrong query", query)
	}
}

func TestCreateTable(t *testing.T) {
	hood, dbName := setupDb(t)
	defer hood.DropDatabase(dbName)

	table := &PgDialectModel{}

	hood.DropTable(table)
	err := hood.CreateTable(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = hood.DropTable(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
}
