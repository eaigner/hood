package hood

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"testing"
	"time"
)

type Sqlite3DialectModel struct {
	Prim   Id
	First  string `sql:"notnull"`
	Last   string `sql:"default('defaultValue')"`
	Amount int
}

func setupDbSqlite(t *testing.T) *Hood {
	os.Remove("/tmp/foo.db")
	hood, err := Open("sqlite3", "/tmp/foo.db")
	if err != nil {
		t.Fatal("could not open db", err)
	}
	hood.Log = true

	return hood
}

func TestTransactionSqlite(t *testing.T) {
	hood := setupDbSqlite(t)

	type sqltTxModel struct {
		Id Id
		A  string
	}

	table := sqltTxModel{
		A: "A",
	}

	hood.DropTable(&table)
	err := hood.CreateTable(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	tx := hood.Begin()
	if _, ok := hood.qo.(*sql.DB); !ok {
		t.Fatal("wrong type")
	}
	if _, ok := tx.qo.(*sql.Tx); !ok {
		t.Fatal("wrong type")
	}
	_, err = tx.Save(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = tx.Rollback()
	if err != nil {
		t.Fatal("error not nil", err)
	}

	var out []sqltTxModel
	err = hood.Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x > 0 {
		t.Fatal("wrong length", x)
	}

	tx = hood.Begin()
	table.Id = 0 // force insert by resetting id
	_, err = tx.Save(&table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal("error not nil", err)
	}

	out = nil
	err = hood.Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 1 {
		t.Fatal("wrong length", x)
	}
}

func TestSqlite3SaveAndDeleteSqlite(t *testing.T) {
	hood := setupDbSqlite(t)

	type sqltSaveModel struct {
		Id Id
		A  string
		B  int
	}
	model1 := sqltSaveModel{
		A: "banana",
		B: 5,
	}
	model2 := sqltSaveModel{
		A: "orange",
		B: 4,
	}

	hood.DropTable(&model1)

	err := hood.CreateTable(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	id, err := hood.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	model1.A = "grape"
	model1.B = 9

	id, err = hood.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	id, err = hood.Save(&model2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 2 {
		t.Fatal("wrong id", id)
	}
	if model2.Id != id {
		t.Fatal("id should have been copied", model2.Id)
	}

	id2, err := hood.Delete(&model2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != id2 {
		t.Fatal("wrong id", id, id2)
	}
}

func TestSqlite3SaveAllDeleteAllSqlite(t *testing.T) {
	type sqltSaveAllDeleteAll struct {
		Id Id
		A  string
	}

	hood := setupDbSqlite(t)
	hood.DropTable(&sqltSaveAllDeleteAll{})

	models := []sqltSaveAllDeleteAll{
		sqltSaveAllDeleteAll{A: "A"},
		sqltSaveAllDeleteAll{A: "B"},
	}
	err := hood.CreateTable(&sqltSaveAllDeleteAll{})
	if err != nil {
		t.Fatal("error not nil", err)
	}

	ids, err := hood.SaveAll(&models)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(ids); x != 2 {
		t.Fatal("wrong id count", x)
	}
	if x := ids[0]; x != 1 {
		t.Fatal("wrong id", x)
	}
	if x := ids[1]; x != 2 {
		t.Fatal("wrong id", x)
	}
	if x := models[0].Id; x != 1 {
		t.Fatal("wrong id", x)
	}
	if x := models[1].Id; x != 2 {
		t.Fatal("wrong id", x)
	}

	_, err = hood.DeleteAll(&models)
	if err != nil {
		t.Fatal("error not nil", err)
	}
}

func TestSqlite3FindSqlite(t *testing.T) {
	hood := setupDbSqlite(t)
	now := time.Now()

	type sqltFindModel struct {
		Id Id
		A  string
		B  int
		C  int8
		D  int16
		E  int32
		F  int64
		G  uint
		H  uint8
		I  uint16
		J  uint32
		K  uint64
		L  float32
		M  float64
		N  []byte
		O  VarChar
		P  time.Time
	}
	model1 := sqltFindModel{
		A: "string!",
		B: -1,
		C: -2,
		D: -3,
		E: -4,
		F: -5,
		G: 6,
		H: 7,
		I: 8,
		J: 9,
		K: 10,
		L: 11.5,
		M: 12.6,
		N: []byte("bytes!"),
		O: "varchar!",
		P: now,
	}

	hood.DropTable(&model1)

	err := hood.CreateTable(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}

	var out []sqltFindModel
	err = hood.Where("a = ? AND j = ?", "string!", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if out != nil {
		t.Fatal("output should be nil", out)
	}

	id, err := hood.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 1 {
		t.Fatal("wrong id", id)
	}

	err = hood.Where("a = ? AND j = ?", "string!", 9).Find(&out)
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
		if x := v.Id; x != 1 {
			t.Fatal("invalid value", x)
		}
		if x := v.A; x != "string!" {
			t.Fatal("invalid value", x)
		}
		if x := v.B; x != -1 {
			t.Fatal("invalid value", x)
		}
		if x := v.C; x != -2 {
			t.Fatal("invalid value", x)
		}
		if x := v.D; x != -3 {
			t.Fatal("invalid value", x)
		}
		if x := v.E; x != -4 {
			t.Fatal("invalid value", x)
		}
		if x := v.F; x != -5 {
			t.Fatal("invalid value", x)
		}
		if x := v.G; x != 6 {
			t.Fatal("invalid value", x)
		}
		if x := v.H; x != 7 {
			t.Fatal("invalid value", x)
		}
		if x := v.I; x != 8 {
			t.Fatal("invalid value", x)
		}
		if x := v.J; x != 9 {
			t.Fatal("invalid value", x)
		}
		if x := v.K; x != 10 {
			t.Fatal("invalid value", x)
		}
		if x := v.L; x != 11.5 {
			t.Fatal("invalid value", x)
		}
		if x := v.M; x != 12.6 {
			t.Fatal("invalid value", x)
		}
		if x := v.N; string(x) != "bytes!" {
			t.Fatal("invalid value", x)
		}
		if x := v.O; string(x) != "varchar!" {
			t.Fatal("invalid value", x)
		}
		if x := v.P; now.Equal(x) {
			t.Fatal("invalid value", x)
		}
	}

	model1.Id = 0 // force insert, would update otherwise
	model1.A = "row2"

	id, err = hood.Save(&model1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if id != 2 {
		t.Fatal("wrong id", id)
	}

	out = nil
	err = hood.Where("a = ? AND j = ?", "row2", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 1 {
		t.Fatal("invalid output length", x)
	}

	out = nil
	err = hood.Where("j = ?", 9).Find(&out)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if x := len(out); x != 2 {
		t.Fatal("invalid output length", x)
	}
}

func TestSqlTypeSqlite(t *testing.T) {
	d := &Sqlite{}
	if x := d.SqlType(true, 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	var indirect interface{} = true
	if x := d.SqlType(indirect, 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(uint32(2), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(Id(1), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(int64(1), 0); x != "integer" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(1.8, 0); x != "real" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType([]byte("asdf"), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType("astring", 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("a"), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(VarChar("b"), 128); x != "text" {
		t.Fatal("wrong type", x)
	}
	if x := d.SqlType(time.Now(), 0); x != "text" {
		t.Fatal("wrong type", x)
	}
}

func TestCreateTableSqlSqlite(t *testing.T) {
	hood := New(nil, &Sqlite{})
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
	model, err := interfaceToModel(table)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	query := hood.createTableSql(model)
	if query != `CREATE TABLE without_pk ( first text, last text, amount integer )` {
		t.Fatal("wrong query", query)
	}
	type withPk struct {
		Primary Id
		First   string
		Last    string
		Amount  int
	}
	table2 := &withPk{
		First:  "erik",
		Last:   "aigner",
		Amount: 5,
	}
	model, err = interfaceToModel(table2)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	query = hood.createTableSql(model)
	if query != `CREATE TABLE with_pk ( primary integer PRIMARY KEY AUTOINCREMENT, first text, last text, amount integer )` {
		t.Fatal("wrong query", query)
	}
}

func TestCreateTableSqlite(t *testing.T) {
	hood := setupDbSqlite(t)

	table := &Sqlite3DialectModel{}

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
