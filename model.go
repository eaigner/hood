package hood

import (
	"errors"
	"reflect"
)

func snakeCaseName(i interface{}) string {
	return snakeCase(reflect.TypeOf(i).Elem().Name())
}

// TODO: move to mapper.go
func interfaceToModel(f interface{}, dialect Dialect) (*Model, error) {
	v := reflect.Indirect(reflect.ValueOf(f))
	if v.Kind() != reflect.Struct {
		return nil, errors.New("model is not a struct")
	}
	t := v.Type()
	m := &Model{
		Pk:     nil,
		Table:  snakeCaseName(f),
		Fields: []*Field{},
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		isPk := field.Type.Name() == reflect.TypeOf(Id(0)).Name()
		fd := &Field{
			Name:    snakeCase(field.Name),
			Value:   v.FieldByName(field.Name).Interface(),
			NotNull: (field.Tag.Get("notnull") == "true"),
			Default: field.Tag.Get("default"),
		}
		if isPk {
			m.Pk = fd
		}
		m.Fields = append(m.Fields, fd)
	}
	return m, nil
}
