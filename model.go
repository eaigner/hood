package hood

import (
	"errors"
	"reflect"
)

func modelFieldOrTableName(i interface{}) string {
	return snakeCase(reflect.TypeOf(i).Elem().Name())
}

func modelMap(model interface{}) (Model, *Pk, error) {
	v := reflect.Indirect(reflect.ValueOf(model))
	if v.Kind() != reflect.Struct {
		return nil, nil, errors.New("model is not a struct")
	}
	t := v.Type()
	m := make(Model)
	pk := (*Pk)(nil)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag == "PK" {
			pk = &Pk{
				Name: snakeCase(string(field.Name)),
				Type: field.Type,
			}
		}
		key := snakeCase(field.Name)
		m[key] = v.FieldByName(field.Name).Interface()
	}
	return m, pk, nil
}
