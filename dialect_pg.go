package hood

import (
	"fmt"
	"reflect"
	"strings"
)

type DialectPg struct{}

func (d *DialectPg) Name() string {
	return "postgres"
}

func (d *DialectPg) Pk() string {
	return "id"
}

func (d *DialectPg) Quote(s string) string {
	q := `"`
	return strings.Join([]string{q, s, q}, "")
}

func (d *DialectPg) MarkerStartPos() int {
	return 1
}

func (d *DialectPg) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos)
}

func (d *DialectPg) SqlType(f interface{}, size int, autoIncr bool) string {
	t := reflect.TypeOf(f)
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		if autoIncr {
			return "serial"
		}
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if autoIncr {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double precision"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
	case reflect.String:
		return "text"
	}
	if size < 1 {
		size = 255
	}
	return fmt.Sprintf("varchar(%d)", size)
}
