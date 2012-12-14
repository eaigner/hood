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
	m, pk, err := modelMap(model)
	if err != nil {
		t.Fatal("error not nil", err)
	}
	if pk == nil {
		t.Fatal("pk nil")
	}
	if pk.Name != "prim_key" {
		t.Fatal("wrong pk name", pk.Name)
	}
	if pk.Type.Kind() != reflect.Int {
		t.Fatal("wrong pk type", pk.Type.String())
	}
	if x := len(m); x != 4 {
		t.Fatal("should have 4 fields, has", x)
	}
	if x := m["prim_key"]; x != 6 {
		t.Fatal("wrong primary key", x)
	}
	if x := m["first_name"]; x != "Erik" {
		t.Fatal("wrong first name", x)
	}
	if x := m["last_name"]; x != "Aigner" {
		t.Fatal("wrong last name", x)
	}
	if x := m["address"]; x != "Nowhere 7" {
		t.Fatal("wrong address", x)
	}
}
