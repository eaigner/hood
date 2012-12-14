package hood

import (
	"testing"
)

type sampleModel struct {
	Prim   string `pk:"true"auto:"false"`
	First  string `null:"true"`
	Last   string `default:"DEFVAL"`
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
	if sql != `INSERT INTO sample_model (first, last, amount) VALUES ($1, $2, $3)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestUpdateSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := interfaceToModel(data1)
	sql := hood.updateSql(model)
	if sql != `UPDATE sample_model (first, last, amount) VALUES ($1, $2, $3) WHERE prim = $4` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestDeleteSQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	model, _ := interfaceToModel(data1)
	sql := hood.deleteSql(model)
	if sql != `DELETE FROM sample_model WHERE prim = $1` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestSubstituteMarkers(t *testing.T) {
	hood := New(nil, &DialectPg{})
	s := hood.substituteMarkers("name = ?")
	if s != "name = $1" {
		t.Fatalf("wrong substitution: '%v'", s)
	}
	hood.Reset()
	s = hood.substituteMarkers("name = ?, balance = ?")
	if s != "name = $1, balance = $2" {
		t.Fatalf("wrong substitution: '%v'", s)
	}
}

func TestQuerySQL(t *testing.T) {
	hood := New(nil, &DialectPg{})
	hood.Select("*", &sampleModel{})
	hood.Where("id = ?", "erik")
	hood.Join("INNER", "orders", "users.id == orders.id")
	hood.GroupBy("name")
	hood.Having("SUM(price) < 2000")
	hood.OrderBy("first_name")
	hood.Offset(3)
	hood.Limit(10)
	sql := hood.querySql()
	if sql != `SELECT * FROM sample_model INNER JOIN orders ON users.id == orders.id WHERE id = $1 GROUP BY name HAVING SUM(price) < 2000 ORDER BY first_name LIMIT 10 OFFSET 3` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestQuerySQLWhere(t *testing.T) {
	hood := New(nil, &DialectPg{})
	hood.Select("*", &sampleModel{})
	hood.Where("name = ?", "erik")
	sql := hood.querySql()
	if sql != `SELECT * FROM sample_model WHERE name = $1` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
	hood.Reset()
	hood.Select("*", &sampleModel{})
	hood.Where(3)
	sql = hood.querySql()
	if sql != `SELECT * FROM sample_model WHERE id = $1` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}
