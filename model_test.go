package hood

import (
	"reflect"
	"testing"
)

type SampleModel struct {
	PrimKey   int `PK`
	FirstName string
	LastName  string
	Address   string
}

func TestModelFieldOrTableName(t *testing.T) {
	name := modelFieldOrTableName(&SampleModel{})
	if name != "sample_model" {
		t.Fatal("wrong table name", name)
	}
}

func TestModelMap(t *testing.T) {
	model := &SampleModel{
		PrimKey:   6,
		FirstName: "Erik",
		LastName:  "Aigner",
		Address:   "Nowhere 7",
	}
	m, err := modelMap(model)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if m.Pk == nil {
		t.Fatal("pk nil")
	}
	if m.Pk.Name != "prim_key" {
		t.Fatal("wrong pk name", m.Pk.Name)
	}
	if m.Pk.Type.Kind() != reflect.Int {
		t.Fatal("wrong pk type", m.Pk.Type.String())
	}
	if x := len(m.Fields); x != 4 {
		t.Fatal("should have 4 fields, has", x)
	}
	if x := m.Fields["prim_key"]; x != 6 {
		t.Fatal("wrong primary key", x)
	}
	if x := m.Fields["first_name"]; x != "Erik" {
		t.Fatal("wrong first name", x)
	}
	if x := m.Fields["last_name"]; x != "Aigner" {
		t.Fatal("wrong last name", x)
	}
	if x := m.Fields["address"]; x != "Nowhere 7" {
		t.Fatal("wrong address", x)
	}
}
