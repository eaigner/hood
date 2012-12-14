package hood

import (
	"errors"
	"reflect"
)

func snakeCaseName(i interface{}) string {
	return snakeCase(reflect.TypeOf(i).Elem().Name())
}

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
		isPk := field.Tag.Get("pk") == "true"
		fd := &Field{
			Pk:      isPk,
			Name:    snakeCase(field.Name),
			Value:   v.FieldByName(field.Name).Interface(),
			NotNull: (field.Tag.Get("notnull") == "true"),
			Auto:    (field.Tag.Get("auto") == "true"),
			Default: field.Tag.Get("default"),
		}
		if isPk {
			m.Pk = fd
		}
		m.Fields = append(m.Fields, fd)
	}
	// if a primary key wasn't specified, add implicitly
	if m.Pk == nil {
		m.Pk = &Field{
			Pk:    true,
			Name:  dialect.Pk(),
			Auto:  true,
			Value: int(0),
		}
		m.Fields = append([]*Field{m.Pk}, m.Fields...)
	}
	return m, nil
}
