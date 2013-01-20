package hood

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Base struct {
	Dialect Dialect
}

func (d *Base) NextMarker(pos *int) string {
	m := fmt.Sprintf("$%d", *pos+1)
	*pos++
	return m
}

func (d *Base) Quote(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func (d *Base) ParseBool(value reflect.Value) bool {
	return value.Bool()
}

func (d *Base) SetModelValue(driverValue, fieldValue reflect.Value) error {
	switch fieldValue.Type().Kind() {
	case reflect.Bool:
		fieldValue.SetBool(d.Dialect.ParseBool(driverValue.Elem()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(driverValue.Elem().Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// reading uint from int value causes panic
		switch driverValue.Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue.SetUint(uint64(driverValue.Elem().Int()))
		default:
			fieldValue.SetUint(driverValue.Elem().Uint())
		}
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(driverValue.Elem().Float())
	case reflect.String:
		fieldValue.SetString(string(driverValue.Elem().Bytes()))
	case reflect.Slice:
		if reflect.TypeOf(driverValue.Interface()).Elem().Kind() == reflect.Uint8 {
			fieldValue.SetBytes(driverValue.Elem().Bytes())
		}
	case reflect.Struct:
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			fieldValue.Set(driverValue.Elem())
		}
	}
	return nil
}

func (d *Base) ConvertHoodType(f interface{}) interface{} {
	if t, ok := f.(Created); ok {
		return t.Time
	}
	if t, ok := f.(Updated); ok {
		return t.Time
	}
	return f
}

func (d *Base) QuerySql(hood *Hood) (string, []interface{}) {
	query := make([]string, 0, 20)
	args := make([]interface{}, 0, 20)
	if hood.selectTable != "" {
		selector := "*"
		if c := hood.selectCols; len(c) > 0 {
			quotedCols := make([]string, 0, len(hood.selectCols))
			for _, c := range hood.selectCols {
				quotedCols = append(quotedCols, d.Dialect.Quote(c))
			}
			selector = strings.Join(quotedCols, ", ")
		}
		query = append(query, fmt.Sprintf("SELECT %v FROM %v", selector, d.Dialect.Quote(hood.selectTable)))
	}
	for i, op := range hood.joinOps {
		joinType := "INNER"
		switch op {
		case LeftJoin:
			joinType = "LEFT"
		case RightJoin:
			joinType = "RIGHT"
		case FullJoin:
			joinType = "FULL"
		}
		query = append(query, fmt.Sprintf(
			"%s JOIN %v ON %s.%s = %s.%s",
			joinType,
			d.Dialect.Quote(tableName(hood.joinTables[i])),
			d.Dialect.Quote(tableName(hood.selectTable)),
			d.Dialect.Quote(hood.joinCol1[i]),
			d.Dialect.Quote(tableName(hood.joinTables[i])),
			d.Dialect.Quote(hood.joinCol2[i]),
		),
		)
	}
	if x := hood.whereClauses; len(x) > 0 {
		query = append(query, fmt.Sprintf("WHERE %v", strings.Join(x, " AND ")))
		args = append(args, hood.whereArgs...)
	}
	if x := hood.groupBy; x != "" {
		query = append(query, fmt.Sprintf("GROUP BY %v", d.Dialect.Quote(x)))
	}
	if x := hood.havingCond; x != "" {
		query = append(query, fmt.Sprintf("HAVING %v", x))
		args = append(args, hood.havingArgs...)
	}
	if x := hood.orderBy; x != "" {
		query = append(query, fmt.Sprintf("ORDER BY %v", d.Dialect.Quote(x)))
	}
	if x := hood.limit; x > 0 {
		query = append(query, "LIMIT ?")
		args = append(args, hood.limit)
	}
	if x := hood.offset; x > 0 {
		query = append(query, "OFFSET ?")
		args = append(args, hood.offset)
	}
	return hood.substituteMarkers(strings.Join(query, " ")), args
}

func (d *Base) Insert(hood *Hood, model *Model) (Id, error) {
	sql, args := d.Dialect.InsertSql(model)
	result, err := hood.Exec(sql, args...)
	if err != nil {
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return Id(id), nil
}

func (d *Base) InsertSql(model *Model) (string, []interface{}) {
	m := 0
	columns, markers, values := columnsMarkersAndValuesForModel(d.Dialect, model, &m)
	quotedColumns := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.Dialect.Quote(c))
	}
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v)",
		d.Dialect.Quote(model.Table),
		strings.Join(quotedColumns, ", "),
		strings.Join(markers, ", "),
	)
	return sql, values
}

func (d *Base) Update(hood *Hood, model *Model) (Id, error) {
	sql, args := d.Dialect.UpdateSql(model)
	_, err := hood.Exec(sql, args...)
	if err != nil {
		return -1, err
	}
	return model.Pk.Value.(Id), nil
}

func (d *Base) UpdateSql(model *Model) (string, []interface{}) {
	m := 0
	columns, markers, values := columnsMarkersAndValuesForModel(d.Dialect, model, &m)
	pairs := make([]string, 0, len(columns))
	for i, column := range columns {
		pairs = append(pairs, fmt.Sprintf("%v = %v", d.Dialect.Quote(column), markers[i]))
	}
	sql := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %v",
		d.Dialect.Quote(model.Table),
		strings.Join(pairs, ", "),
		d.Dialect.Quote(model.Pk.Name),
		d.Dialect.NextMarker(&m),
	)
	values = append(values, model.Pk.Value)
	return sql, values
}

func (d *Base) Delete(hood *Hood, model *Model) (Id, error) {
	sql, args := d.Dialect.DeleteSql(model)
	_, err := hood.Exec(sql, args...)
	return args[0].(Id), err
}

func (d *Base) DeleteSql(model *Model) (string, []interface{}) {
	n := 0
	return fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %v",
		d.Dialect.Quote(model.Table),
		d.Dialect.Quote(model.Pk.Name),
		d.Dialect.NextMarker(&n),
	), []interface{}{model.Pk.Value}
}

func (d *Base) CreateTable(hood *Hood, model *Model) error {
	_, err := hood.Exec(d.Dialect.CreateTableSql(model, false))
	return err
}

func (d *Base) CreateTableIfNotExists(hood *Hood, model *Model) error {
	_, err := hood.Exec(d.Dialect.CreateTableSql(model, true))
	return err
}

func (d *Base) CreateTableSql(model *Model, ifNotExists bool) string {
	a := []string{"CREATE TABLE "}
	if ifNotExists {
		a = append(a, "IF NOT EXISTS ")
	}
	a = append(a, d.Dialect.Quote(model.Table), " ( ")
	for i, field := range model.Fields {
		b := []string{
			d.Dialect.Quote(field.Name),
			d.Dialect.SqlType(field.Value, field.Size()),
		}
		if field.NotNull() {
			b = append(b, d.Dialect.KeywordNotNull())
		}
		if x := field.Default(); x != "" {
			b = append(b, d.Dialect.KeywordDefault(x))
		}
		if field.PrimaryKey() {
			b = append(b, d.Dialect.KeywordPrimaryKey())
		}
		if incKeyword := d.Dialect.KeywordAutoIncrement(); field.PrimaryKey() && incKeyword != "" {
			b = append(b, incKeyword)
		}
		a = append(a, strings.Join(b, " "))
		if i < len(model.Fields)-1 {
			a = append(a, ", ")
		}
	}
	a = append(a, " )")
	return strings.Join(a, "")
}

func (d *Base) DropTable(hood *Hood, table string) error {
	_, err := hood.Exec(d.Dialect.DropTableSql(table, false))
	return err
}

func (d *Base) DropTableIfExists(hood *Hood, table string) error {
	_, err := hood.Exec(d.Dialect.DropTableSql(table, true))
	return err
}

func (d *Base) DropTableSql(table string, ifExists bool) string {
	a := []string{"DROP TABLE"}
	if ifExists {
		a = append(a, "IF EXISTS")
	}
	a = append(a, d.Dialect.Quote(table))
	return strings.Join(a, " ")
}

func (d *Base) RenameTable(hood *Hood, from, to string) error {
	_, err := hood.Exec(d.Dialect.RenameTableSql(from, to))
	return err
}

func (d *Base) RenameTableSql(from, to string) string {
	return fmt.Sprintf("ALTER TABLE %v RENAME TO %v", d.Dialect.Quote(from), d.Dialect.Quote(to))
}

func (d *Base) AddColumn(hood *Hood, table, column string, typ interface{}, size int) error {
	_, err := hood.Exec(d.Dialect.AddColumnSql(table, column, typ, size))
	return err
}

func (d *Base) AddColumnSql(table, column string, typ interface{}, size int) string {
	return fmt.Sprintf(
		"ALTER TABLE %v ADD COLUMN %v %v",
		d.Dialect.Quote(table),
		d.Dialect.Quote(column),
		d.Dialect.SqlType(typ, size),
	)
}

func (d *Base) RenameColumn(hood *Hood, table, from, to string) error {
	_, err := hood.Exec(d.Dialect.RenameColumnSql(table, from, to))
	return err
}

func (d *Base) RenameColumnSql(table, from, to string) string {
	return fmt.Sprintf(
		"ALTER TABLE %v RENAME COLUMN %v TO %v",
		d.Dialect.Quote(table),
		d.Dialect.Quote(from),
		d.Dialect.Quote(to),
	)
}

func (d *Base) ChangeColumn(hood *Hood, table, column string, typ interface{}, size int) error {
	_, err := hood.Exec(d.Dialect.ChangeColumnSql(table, column, typ, size))
	return err
}

func (d *Base) ChangeColumnSql(table, column string, typ interface{}, size int) string {
	return fmt.Sprintf(
		"ALTER TABLE %v ALTER COLUMN %v TYPE %v",
		d.Dialect.Quote(table),
		d.Dialect.Quote(column),
		d.Dialect.SqlType(typ, size),
	)
}

func (d *Base) DropColumn(hood *Hood, table, column string) error {
	_, err := hood.Exec(d.Dialect.DropColumnSql(table, column))
	return err
}

func (d *Base) DropColumnSql(table, column string) string {
	return fmt.Sprintf(
		"ALTER TABLE %v DROP COLUMN %v",
		d.Dialect.Quote(table),
		d.Dialect.Quote(column),
	)
}

func (d *Base) CreateIndex(hood *Hood, name, table string, unique bool, columns ...string) error {
	_, err := hood.Exec(d.Dialect.CreateIndexSql(name, table, unique, columns...))
	return err
}

func (d *Base) CreateIndexSql(name, table string, unique bool, columns ...string) string {
	a := []string{"CREATE"}
	if unique {
		a = append(a, "UNIQUE")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, c := range columns {
		quotedColumns = append(quotedColumns, d.Dialect.Quote(c))
	}
	a = append(a, fmt.Sprintf(
		"INDEX %v ON %v (%v)",
		d.Dialect.Quote(name),
		d.Dialect.Quote(table),
		strings.Join(quotedColumns, ", "),
	))
	return strings.Join(a, " ")
}

func (d *Base) DropIndex(hood *Hood, name string) error {
	_, err := hood.Exec(d.Dialect.DropIndexSql(name))
	return err
}

func (d *Base) DropIndexSql(name string) string {
	return fmt.Sprintf("DROP INDEX %v", d.Dialect.Quote(name))
}

func (d *Base) KeywordNotNull() string {
	return "NOT NULL"
}

func (d *Base) KeywordDefault(s string) string {
	return fmt.Sprintf("DEFAULT %v", s)
}

func (d *Base) KeywordPrimaryKey() string {
	return "PRIMARY KEY"
}

func (d *Base) KeywordAutoIncrement() string {
	return "AUTOINCREMENT"
}
