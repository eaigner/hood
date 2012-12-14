package hood

import (
	"testing"
)

type sampleModel struct {
	Prim   string `PK`
	First  string
	Last   string
	Amount int
}

var data1 *sampleModel = &sampleModel{
	Prim:   "prim",
	First:  "Erik",
	Last:   "Aigner",
	Amount: 0,
}

func TestInsertSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := modelMap(data1)
	sql := hood.insertSql(model)
	if sql != `INSERT INTO "sample_model" ("amount", "first", "last") VALUES ($0, $1, $2)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestUpdateSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := modelMap(data1)
	sql := hood.updateSql(model)
	if sql != `UPDATE "sample_model" ("amount", "first", "last") VALUES ($0, $1, $2) WHERE "prim" = $3` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}
