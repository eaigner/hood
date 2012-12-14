package hood

import (
	"testing"
)

type SampleModel2 struct {
	Prim   string `PK`
	First  string
	Last   string
	Amount int
}

func TestInsertSQL(t *testing.T) {
	m := &SampleModel2{
		Prim:   "prim",
		First:  "Erik",
		Last:   "Aigner",
		Amount: 0,
	}
	hood := New(nil, &DialectPg{})
	model, _ := modelMap(m)
	sql := hood.insertSql(model)
	if sql != `INSERT INTO "sample_model2" ("amount", "first", "last") VALUES ($0, $1, $2)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}
