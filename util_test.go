package hood

import (
	"testing"
)

func TestSnakeToUpperCamelCase(t *testing.T) {
	if s := snakeToUpperCamelCase("table_name"); s != "TableName" {
		t.Fatal("wrong string", s)
	}
}
