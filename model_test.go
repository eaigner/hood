package hood

import (
	"testing"
)

type SampleModel struct {
	PrimKey   Id
	FirstName string `notnull:"true"`
	LastName  string `default:"last"`
	Address   string
}

func TestSnakeCaseName(t *testing.T) {
	name := snakeCaseName(&SampleModel{})
	if name != "sample_model" {
		t.Fatal("wrong table name", name)
	}
}

func TestInterfaceToModel(t *testing.T) {
	model := &SampleModel{
		PrimKey:   6,
		FirstName: "Erik",
		LastName:  "Aigner",
		Address:   "Nowhere 7",
	}
	m, err := interfaceToModel(model, &DialectPg{})
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
