package hood

import (
	"testing"
)

func TestSnakeCaseName(t *testing.T) {
	type SampleModel struct{}
	name := snakeCaseName(&SampleModel{})
	if name != "sample_model" {
		t.Fatal("wrong table name", name)
	}
	name = snakeCaseName(SampleModel{})
	if name != "sample_model" {
		t.Fatal("wrong table name", name)
	}
}

func TestSnakeToUpperCamelCase(t *testing.T) {
	if s := snakeToUpperCamelCase("table_name"); s != "TableName" {
		t.Fatal("wrong string", s)
	}
}
