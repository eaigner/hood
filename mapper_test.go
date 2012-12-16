package hood

import (
	"testing"
)

type sampleModel struct {
	Prim   Id
	First  string `null:"true"`
	Last   string `default:"DEFVAL"`
	Amount int
}

var data1 *sampleModel = &sampleModel{
	Prim:   3,
	First:  "Erik",
	Last:   "Aigner",
	Amount: 0,
}

func TestInsertSQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	model, _ := hood.interfaceToModel(data1)
	sql, _ := hood.insertSql(model)
	if sql != `INSERT INTO sample_model (first, last, amount) VALUES ($1, $2, $3)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestUpdateSQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	model, _ := hood.interfaceToModel(data1)
	query, _ := hood.updateSql(model)
	if query != `UPDATE sample_model SET first = $1, last = $2, amount = $3 WHERE prim = $4` {
		t.Fatalf("invalid sql: '%v'", query)
	}
}

func TestDeleteSQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	model, _ := hood.interfaceToModel(data1)
	query, _ := hood.deleteSql(model)
	if query != `DELETE FROM sample_model WHERE prim = $1` {
		t.Fatalf("invalid sql: '%v'", query)
	}
}

func TestSubstituteMarkers(t *testing.T) {
	hood := New(nil, &Postgres{})
	s := hood.substituteMarkers("name = ?")
	if s != "name = $1" {
		t.Fatalf("wrong substitution: '%v'", s)
	}
	if x := hood.markerPos; x != 1 {
		t.Fatal("wrong arg count", x)
	}
	hood.Reset()
	s = hood.substituteMarkers("name = ?, balance = ?")
	if s != "name = $1, balance = $2" {
		t.Fatalf("wrong substitution: '%v'", s)
	}
	if x := hood.markerPos; x != 2 {
		t.Fatal("wrong arg count", x)
	}
}

func TestQuerySQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	hood.Select("*", &sampleModel{})
	hood.Where("id = ?", "erik")
	hood.Join("INNER", "orders", "users.id == orders.id")
	hood.GroupBy("name")
	hood.Having("SUM(price) < ?", 2000)
	hood.OrderBy("first_name")
	hood.Offset(3)
	hood.Limit(10)
	sql := hood.querySql()
	if sql != `SELECT * FROM sample_model INNER JOIN orders ON users.id == orders.id WHERE id = $1 GROUP BY name HAVING SUM(price) < $2 ORDER BY first_name LIMIT $3 OFFSET $4` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestQuerySQLWhere(t *testing.T) {
	hood := New(nil, &Postgres{})
	hood.Select("*", &sampleModel{})
	hood.Where("name = ?", "erik")
	sql := hood.querySql()
	if sql != `SELECT * FROM sample_model WHERE name = $1` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
	hood.Reset()
	hood.Select("*", &sampleModel{})
	hood.Where("id = ?", 3)
	sql = hood.querySql()
	if sql != `SELECT * FROM sample_model WHERE id = $1` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestInterfaceToModel(t *testing.T) {
	type SampleModel struct {
		PrimKey   Id
		FirstName string `notnull:"true"`
		LastName  string `default:"last"`
		Address   string
	}

	model := &SampleModel{
		PrimKey:   6,
		FirstName: "Erik",
		LastName:  "Aigner",
		Address:   "Nowhere 7",
	}
	hood := New(nil, &Postgres{})
	m, err := hood.interfaceToModel(model)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if m.Pk == nil {
		t.Fatal("pk nil")
	}
	if m.Pk.Name != "prim_key" {
		t.Fatal("wrong pk name", m.Pk.Name)
	}
	if x := len(m.Fields); x != 4 {
		t.Fatal("should have 4 fields, has", x)
	}
	f := m.Fields[0]
	if x, ok := f.Value.(Id); ok && x != 6 {
		t.Fatal("wrong primary key", x)
	}
	f = m.Fields[1]
	if x := f.Value; x != "Erik" {
		t.Fatal("wrong first name", x)
	}
	if f.NotNull != true {
		t.Fatal("should have null tag set")
	}
	f = m.Fields[2]
	if x := f.Value; x != "Aigner" {
		t.Fatal("wrong last name", x)
	}
	if f.Default != "last" {
		t.Fatal("should have default set")
	}
	if x := m.Fields[3].Value; x != "Nowhere 7" {
		t.Fatal("wrong address", x)
	}
}
