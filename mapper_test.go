package hood

import (
	"testing"
	"time"
)

type sampleModel struct {
	Prim   Id
	First  string `sql:"notnull"`
	Last   string `sql:"default('orange')"`
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
	model, _ := interfaceToModel(data1)
	sql, _ := hood.insertSql(model)
	if sql != `INSERT INTO sample_model (first, last, amount) VALUES ($1, $2, $3)` {
		t.Fatalf("invalid sql: '%v'", sql)
	}
}

func TestUpdateSQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	model, _ := interfaceToModel(data1)
	query, _ := hood.updateSql(model)
	if query != `UPDATE sample_model SET first = $1, last = $2, amount = $3 WHERE prim = $4` {
		t.Fatalf("invalid sql: '%v'", query)
	}
}

func TestDeleteSQL(t *testing.T) {
	hood := New(nil, &Postgres{})
	model, _ := interfaceToModel(data1)
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

func TestParseTags(t *testing.T) {
	m := parseTags(`pk`)
	if x, ok := m["pk"]; !ok {
		t.Fatal("wrong value", ok, x)
	}
	m = parseTags(`notnull,default('banana')`)
	if x, ok := m["notnull"]; !ok {
		t.Fatal("wrong value", ok, x)
	}
	if x, ok := m["default"]; !ok || x != "'banana'" {
		t.Fatal("wrong value", x)
	}
}

func TestFieldZero(t *testing.T) {
	field := &Field{}
	field.Value = nil
	if !field.Zero() {
		t.Fatal("should be zero")
	}
	field.Value = 0
	if !field.Zero() {
		t.Fatal("should be zero")
	}
	field.Value = ""
	if !field.Zero() {
		t.Fatal("should be zero")
	}
	field.Value = false
	if !field.Zero() {
		t.Fatal("should be zero")
	}
	field.Value = true
	if field.Zero() {
		t.Fatal("should not be zero")
	}
	field.Value = -1
	if field.Zero() {
		t.Fatal("should not be zero")
	}
	field.Value = 1
	if field.Zero() {
		t.Fatal("should not be zero")
	}
	field.Value = "asdf"
	if field.Zero() {
		t.Fatal("should not be zero")
	}
}

func TestInterfaceToModel(t *testing.T) {
	type table struct {
		ColPrimary    Id
		ColAltPrimary string  `sql:"pk"`
		ColNotNull    string  `sql:"notnull,default('banana')"`
		ColVarChar    VarChar `sql:"size(64)"`
		ColTime       time.Time
	}
	now := time.Now()
	table1 := &table{
		ColPrimary:    6,
		ColAltPrimary: "banana",
		ColVarChar:    "orange",
		ColTime:       now,
	}
	m, err := interfaceToModel(table1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if m.Pk == nil {
		t.Fatal("pk nil")
	}
	if m.Pk.Name != "col_alt_primary" {
		t.Fatal("wrong value", m.Pk.Name)
	}
	if x := len(m.Fields); x != 5 {
		t.Fatal("wrong value", x)
	}
	f := m.Fields[0]
	if x, ok := f.Value.(Id); !ok || x != 6 {
		t.Fatal("wrong value", x)
	}
	if !f.PrimaryKey() {
		t.Fatal("wrong value")
	}
	f = m.Fields[1]
	if x, ok := f.Value.(string); !ok || x != "banana" {
		t.Fatal("wrong value", x)
	}
	if !f.PrimaryKey() {
		t.Fatal("wrong value")
	}
	f = m.Fields[2]
	if x, ok := f.Value.(string); !ok || x != "" {
		t.Fatal("wrong value", x)
	}
	if f.Default() != "'banana'" {
		t.Fatal("should value", f.Default())
	}
	if !f.NotNull() {
		t.Fatal("wrong value")
	}
	f = m.Fields[3]
	if x, ok := f.Value.(VarChar); !ok || x != "orange" {
		t.Fatal("wrong value", x)
	}
	if x := f.Size(); x != 64 {
		t.Fatal("wrong value", x)
	}
	f = m.Fields[4]
	if x, ok := f.Value.(time.Time); !ok || !now.Equal(x) {
		t.Fatal("wrong value", x)
	}
}
