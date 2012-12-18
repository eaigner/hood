package hood

import (
	"fmt"
	_ "github.com/bmizerany/pq"
	"reflect"
	"time"
)

func init() {
	RegisterDialect("postgres", &Postgres{})
}

type Postgres struct{}

func (d *Postgres) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos+1)
}

func (d *Postgres) SqlType(f interface{}, size int) string {
	switch f.(type) {
	case Id:
		return "bigserial"
	case VarChar:
		if size < 1 {
			size = 255
		}
		return fmt.Sprintf("varchar(%d)", size)
	case time.Time:
		return "timestamp"
	case bool:
		return "boolean"
	case int, int8, int16, int32, uint, uint8, uint16, uint32:
		return "integer"
	case int64, uint64:
		return "bigint"
	case float32, float64:
		return "double precision"
	case []byte:
		return "bytea"
	case string:
		return "text"
	}
	panic("invalid sql type")
}

func (d *Postgres) ValueToField(value reflect.Value, field reflect.Value) error {
	switch field.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(value.Elem().Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// reading uint from int value causes panic
		switch value.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetUint(uint64(value.Elem().Int()))
		default:
			field.SetUint(value.Elem().Uint())
		}
	case reflect.Float32, reflect.Float64:
		field.SetFloat(value.Elem().Float())
	case reflect.String:
		field.SetString(string(value.Elem().Bytes()))
	case reflect.Slice:
		if reflect.TypeOf(value.Interface()).Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value.Elem().Bytes())
		}
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			field.Set(value.Elem())
		}
	}
	return nil
}

func (d *Postgres) Insert(hood *Hood, model *Model, query string, args ...interface{}) (Id, error, bool) {
	query = fmt.Sprintf("%v RETURNING %v", query, model.Pk.Name)
	var id int64
	err := hood.QueryRow(query, args...).Scan(&id)

	return Id(id), err, true
}

func (d *Postgres) StmtNotNull() string {
	return "NOT NULL"
}

func (d *Postgres) StmtDefault(s string) string {
	return fmt.Sprintf("DEFAULT %v", s)
}

func (d *Postgres) StmtPrimaryKey() string {
	return "PRIMARY KEY"
}

func (d *Postgres) StmtAutoIncrement() string {
	// postgres has not auto increment statement, uses SERIAL type
	return ""
}
