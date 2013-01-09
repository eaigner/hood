package hood

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

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
	field := &ModelField{}
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

func TestFieldValidate(t *testing.T) {
	type Schema struct {
		A string  `validate:"len(3:6)"`
		B int     `validate:"range(10:20)"`
		C VarChar `validate:"len(:4),presence"`
	}
	m, _ := interfaceToModel(&Schema{})
	a := m.Fields[0]
	if x := len(a.ValidateTags); x != 1 {
		t.Fatal("wrong len", x)
	}
	if x, ok := a.ValidateTags["len"]; !ok || x != "3:6" {
		t.Fatal("wrong value", x, ok)
	}
	if err := a.Validate(); err == nil || err.Error() != "value too short" {
		t.Fatal("should not validate")
	}
	a.Value = "abc"
	if err := a.Validate(); err != nil {
		t.Fatal("should validate", err)
	}
	a.Value = "abcdefg"
	if err := a.Validate(); err == nil || err.Error() != "value too long" {
		t.Fatal("should not validate")
	}

	b := m.Fields[1]
	if x := len(b.ValidateTags); x != 1 {
		t.Fatal("wrong len", x)
	}
	if err := b.Validate(); err == nil || err.Error() != "value too small" {
		t.Fatal("should not validate")
	}
	b.Value = 10
	if err := b.Validate(); err != nil {
		t.Fatal("should validate", err)
	}
	b.Value = 21
	if err := b.Validate(); err == nil || err.Error() != "value too big" {
		t.Fatal("should not validate")
	}

	c := m.Fields[2]
	if x := len(c.ValidateTags); x != 2 {
		t.Fatal("wrong len", x)
	}
	if err := c.Validate(); err == nil || err.Error() != "value not set" {
		t.Fatal("should not validate")
	}
	c.Value = "a"
	if err := c.Validate(); err != nil {
		t.Fatal("should validate", err)
	}
	c.Value = "abcde"
	if err := c.Validate(); err == nil || err.Error() != "value too long" {
		t.Fatal("should not validate")
	}
}

func TestFieldOmit(t *testing.T) {
	type Schema struct {
		A string `sql:"-"`
		B string
	}
	m, _ := interfaceToModel(&Schema{})
	if x := len(m.Fields); x != 1 {
		t.Fatal("wrong len", x)
	}
}

type validateSchema struct {
	A string
}

var numValidateFuncCalls = 0

func (v *validateSchema) ValidateX() error {
	numValidateFuncCalls++
	if v.A == "banana" {
		return NewValidationError(1, "value cannot be banana")
	}
	return nil
}

func (v *validateSchema) ValidateY() error {
	numValidateFuncCalls++
	return NewValidationError(2, "ValidateY failed")
}

func TestValidationMethods(t *testing.T) {
	hd := New(nil, &Postgres{})
	m := &validateSchema{}
	err := hd.Validate(m)
	if err == nil || err.Error() != "ValidateY failed" {
		t.Fatal("wrong error", err)
	}
	if v, ok := err.(*ValidationError); !ok {
		t.Fatal("should be of type ValidationError", v)
	}
	if numValidateFuncCalls != 2 {
		t.Fatal("should have called validation func")
	}
	numValidateFuncCalls = 0
	m.A = "banana"
	err = hd.Validate(m)
	if err == nil || err.Error() != "value cannot be banana" {
		t.Fatal("wrong error", err)
	}
	if numValidateFuncCalls != 1 {
		t.Fatal("should have called validation func")
	}
}

func TestInterfaceToModelWithEmbedded(t *testing.T) {
	type embed struct {
		Name  string
		Value string
	}
	type table struct {
		ColPrimary Id
		embed
	}
	table1 := &table{
		6, embed{"Mrs. A", "infinite"},
	}
	m, err := interfaceToModel(table1)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	f := m.Fields[1]
	if x, ok := f.Value.(string); !ok || x != "Mrs. A" {
		t.Fatal("wrong value from embedded struct")
	}
}

func TestInterfaceToModel(t *testing.T) {
	type table struct {
		ColPrimary    Id
		ColAltPrimary string  `sql:"pk"`
		ColNotNull    string  `sql:"notnull,default('banana')"`
		ColVarChar    VarChar `sql:"size(64)"`
		ColTime       time.Time
		MyIndex       Index       `sql:"columns(col_primary:col_time)"`
		MyUniqueIndex UniqueIndex `sql:"columns(col_var_char:col_time)"`
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
	if x := len(m.Indexes); x != 2 {
		t.Fatal("wrong value", x)
	}
	if x := m.Indexes[0].Name; x != "my_index" {
		t.Fatal("wrong index name", x)
	}
	if x := m.Indexes[0].Columns; strings.Join(x, ":") != "col_primary:col_time" {
		t.Fatal("wrong index columns", x)
	}
	if x := m.Indexes[0].Unique; x != false {
		t.Fatal("wrong index uniqueness", x)
	}
	if x := m.Indexes[1].Name; x != "my_unique_index" {
		t.Fatal("wrong index name", x)
	}
	if x := m.Indexes[1].Columns; strings.Join(x, ":") != "col_var_char:col_time" {
		t.Fatal("wrong index columns", x)
	}
	if x := m.Indexes[1].Unique; x != true {
		t.Fatal("wrong index uniqueness", x)
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

func TestSchemaGeneration(t *testing.T) {
	hd := New(nil, NewPostgres())
	hd.dryRun = true
	if x := len(hd.schema); x != 0 {
		t.Fatal("invalid schema state", x)
	}
	type Users struct {
		Id        Id
		First     string
		Last      string
		NameIndex UniqueIndex `sql:"columns(first:last)"`
	}
	hd.CreateTable(&Users{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Indexes); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Fields); x != 3 {
		t.Fatal("invalid schema state", x)
	}
	type DropMe struct {
		Id Id
	}
	hd.CreateTable(&DropMe{})
	if x := len(hd.schema); x != 2 {
		t.Fatal("invalid schema state", x)
	}
	hd.DropTable(&DropMe{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Table; x != "users" {
		t.Fatal("invalid schema state", x)
	}
	hd.RenameTable(&Users{}, "customers")
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Table; x != "customers" {
		t.Fatal("invalid schema state", x)
	}
	hd.AddColumns("customers", struct {
		Balance int
	}{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Fields); x != 4 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[3].Name; x != "balance" {
		t.Fatal("invalid schema state", x)
	}
	hd.RenameColumn("customers", "balance", "amount")
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Fields); x != 4 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[3].Name; x != "amount" {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[3].Value; reflect.TypeOf(x) != reflect.TypeOf(int(0)) {
		t.Fatal("invalid schema state", x)
	}
	hd.ChangeColumns("customers", struct {
		Amount string
	}{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Fields); x != 4 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[3].Name; x != "amount" {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[3].Value; reflect.TypeOf(x) != reflect.TypeOf("") {
		t.Fatal("invalid schema state", x)
	}
	hd.RemoveColumns("customers", struct {
		First string
		Last  string
	}{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Fields); x != 2 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[0].Name; x != "id" {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Fields[1].Name; x != "amount" {
		t.Fatal("invalid schema state", x)
	}
	hd.CreateIndex("my_index", "customers", struct {
		AmountIndex Index `sql:"columns(amount)"`
	}{})
	if x := len(hd.schema); x != 1 {
		t.Fatal("invalid schema state", x)
	}
	if x := len(hd.schema[0].Indexes); x != 2 {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Indexes[0].Name; x != "name_index" {
		t.Fatal("invalid schema state", x)
	}
	if x := hd.schema[0].Indexes[1].Name; x != "amount_index" {
		t.Fatal("invalid schema state", x)
	}
}
