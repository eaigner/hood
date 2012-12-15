package hood

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"testing"
)

const (
	disableLiveTests = true
)

type PgDialectModel struct {
	Prim   int    `pk:"true"auto:"true"`
	First  string `null:"true"`
	Last   string `default:"'defaultValue'"`
	Amount int
}

func setupDb(t *testing.T) *Hood {
	db, err := sql.Open("postgres", "user=hood dbname=hood_test sslmode=disable")
	if err != nil {
		t.Fatal("could not open db", err)
	}
	hood := New(db, &DialectPg{})
	hood.Log = true

	return hood
}

func TestPgSave(t *testing.T) {
	if disableLiveTests {
		return
	}
	hood := setupDb(t)

	type pgSaveModel struct {
		First  string
		Last   string
		Amount int
	}
	model1 := &pgSaveModel{
		"erik",
		"aigner",
		5,
	}
	model2 := &pgSaveModel{
		"markus",
		"schumacher",
		4,
	}

	hood.DropTable(model1)

	err := hood.CreateTable(model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	id, err := hood.Save(model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	id, err = hood.Save(model2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 2 {
		t.Fatal("wrong id", id)
	}
}

func TestPgFind(t *testing.T) {
	if disableLiveTests {
		return
	}
	hood := setupDb(t)

	type pgFindModel struct {
		First  string
		Last   string
		Amount int
	}
	model1 := &pgFindModel{
		"erik",
		"aigner",
		5,
	}

	hood.DropTable(model1)

	err := hood.CreateTable(model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	var out []pgFindModel
	err = hood.Where("first = ? AND amount = ?", "erik", 5).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if out != nil {
		t.Fatal("output should be nil", out)
	}

	id, err := hood.Save(model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	err = hood.Where("first = ? AND amount = ?", "erik", 5).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if out == nil {
		t.Fatal("output should not be nil")
	}
	if x := len(out); x != 1 {
		t.Fatal("invalid output length", x)
	}
	for _, v := range out {
		if x := v.Amount; x != 5 {
			t.Fatal("invalid amount", x)
		}
		if x := v.First; x != "erik" {
			t.Fatal("invalid first", x)
		}
		if x := v.Last; x != "aigner" {
			t.Fatal("invalid last", x)
		}
	}
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
	hood := New(nil, &DialectPg{})
	type withoutPk struct {
		First  string
		Last   string
		Amount int
	}
	table := &withoutPk{
		"erik",
		"aigner",
		5,
	}
	model, err := interfaceToModel(table, hood.Dialect)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	query := hood.createTableSql(model)
	if query != `CREATE TABLE without_pk ( id serial PRIMARY KEY, first text, last text, amount integer )` {
		t.Fatal("wrong query", query)
	}
	type withPk struct {
		Primary int `pk:"true"auto:"true"`
		First   string
		Last    string
		Amount  int
	}
	table2 := &withPk{
		First:  "erik",
		Last:   "aigner",
		Amount: 5,
	}
	model, err = interfaceToModel(table2, hood.Dialect)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	query = hood.createTableSql(model)
	if query != `CREATE TABLE with_pk ( primary serial PRIMARY KEY, first text, last text, amount integer )` {
		t.Fatal("wrong query", query)
	}
}

func TestCreateTable(t *testing.T) {
	if disableLiveTests {
		return
	}
	hood := setupDb(t)

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
