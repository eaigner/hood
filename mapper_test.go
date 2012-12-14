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
	model, _ := interfaceToModel(data1)
	sql := hood.insertSql(model)
	if sql != `INSERT INTO "sample_model" ("amount", "first", "last") VALUES ($0, $1, $2)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestUpdateSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := interfaceToModel(data1)
	sql := hood.updateSql(model)
	if sql != `UPDATE "sample_model" ("amount", "first", "last") VALUES ($0, $1, $2) WHERE "prim" = $3` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestDeleteSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := interfaceToModel(data1)
	sql := hood.deleteSql(model)
	if sql != `DELETE FROM "sample_model" WHERE "prim" = $0` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestQuerySQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	hood.Select("*", &sampleModel{})
	hood.Where(3)
	hood.Join("INNER", "orders", "users.id == orders.id")
	hood.GroupBy("name")
	hood.Having("SUM(price) < 2000")
	hood.OrderBy("first_name")
	hood.Offset(3)
	hood.Limit(10)
	sql := hood.querySql()
	if sql != `SELECT * FROM sample_model INNER JOIN orders ON users.id == orders.id WHERE "id" = $0 GROUP BY name HAVING SUM(price) < 2000 ORDER BY first_name LIMIT 10 OFFSET 3` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}
