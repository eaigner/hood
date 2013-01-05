package hood

import (
	// _ "github.com/mattn/go-sqlite3"
	"reflect"
	"time"
	"unsafe"
)

func init() {
	// It's probably not a good idea to register the dialect by default since
	// it requires specific packages to be installed on the target system!
	// 
	// RegisterDialect("sqlite3", &Sqlite3{})
}

type Sqlite3 struct {
	Base
}

func NewSqlite3() Dialect {
	d := &Sqlite3{}
	d.Base.Dialect = d
	return d
}

func (d *Sqlite3) SqlType(f interface{}, size int) string {
	switch f.(type) {
	case Id:
		return "integer"
	case VarChar:
		return "text"
	case time.Time:
		return "text"
	case bool:
		// 0 or 1
		return "integer"
	case int, int8, int16, int32, uint, uint8, uint16, uint32, int64, uint64:
		return "integer"
	case float32, float64:
		return "real"
	case []byte:
		return "text"
	case string:
		return "text"
	}
	panic("invalid sql type")
}

func (d *Sqlite3) ValueToField(value reflect.Value, field reflect.Value) error {
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
	case reflect.Bool:
		if value.Elem().Int() == 0 {
			field.SetBool(false)
		} else {
			field.SetBool(true)
		}
	case reflect.Float32, reflect.Float64:
		field.SetFloat(value.Elem().Float())
	case reflect.String:
		field.SetString(value.Elem().String())
	case reflect.Slice:
		if reflect.TypeOf(value.Interface()).Elem().Kind() == reflect.Uint8 {
			field.SetBytes(value.Elem().Bytes())
		}
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse("2006-01-02 15:04:05", value.Elem().String())
			if err != nil {
				return err
			}
			v := reflect.NewAt(reflect.TypeOf(time.Time{}), unsafe.Pointer(&t))
			field.Set(v.Elem())
		}
	}
	return nil
}

func (d *Sqlite3) RenameColumn(hood *Hood, table, from, to string) error {
	// TODO: implement column rename via table copy, see
	// http://stackoverflow.com/questions/805363/how-do-i-rename-a-column-in-a-sqlite-database-table
	panic("not implemented")
}

func (d *Sqlite3) RenameColumnSql(table, from, to string) string {
	return ""
}

func (d *Sqlite3) ChangeColumn(hood *Hood, table, column string, typ interface{}, size int) error {
	// TODO: implement column change via table copy, see
	// http://stackoverflow.com/questions/805363/how-do-i-rename-a-column-in-a-sqlite-database-table
	panic("not implemented")
}

func (d *Sqlite3) ChangeColumnSql(table, column string, typ interface{}, size int) string {
	return ""
}
