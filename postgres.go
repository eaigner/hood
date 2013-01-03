package hood

import (
	"fmt"
	_ "github.com/bmizerany/pq"
	"reflect"
	"strings"
	"time"
)

func init() {
	RegisterDialect("postgres", &Postgres{})
}

type Postgres struct{}

func (d *Postgres) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos+1)
}

func (d *Postgres) NextMarker(pos *int) string {
	m := d.Marker(*pos)
	*pos++
	return m
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

func (d *Postgres) QuerySql(hood *Hood) (string, []interface{}) {
	query := make([]string, 0, 20)
	args := make([]interface{}, 0, 20)
	if hood.selectCols != "" && hood.selectTable != "" {
		query = append(query, fmt.Sprintf("SELECT %v FROM %v", hood.selectCols, hood.selectTable))
	}
	for i, op := range hood.joinOps {
		query = append(query, fmt.Sprintf("%v JOIN %v ON %v", op, hood.joinTables[i], hood.joinCond[i]))
	}
	if x := hood.whereClauses; len(x) > 0 {
		query = append(query, fmt.Sprintf("WHERE %v", strings.Join(x, " AND ")))
		args = append(args, hood.whereArgs...)
	}
	if x := hood.groupBy; x != "" {
		query = append(query, fmt.Sprintf("GROUP BY %v", x))
	}
	if x := hood.havingCond; x != "" {
		query = append(query, fmt.Sprintf("HAVING %v", x))
		args = append(args, hood.havingArgs...)
	}
	if x := hood.orderBy; x != "" {
		query = append(query, fmt.Sprintf("ORDER BY %v", x))
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

func (d *Postgres) columnsMarkersAndValuesForModel(model *Model, markerPos *int) ([]string, []string, []interface{}) {
	columns := make([]string, 0, len(model.Fields))
	markers := make([]string, 0, len(columns))
	values := make([]interface{}, 0, len(columns))
	for _, column := range model.Fields {
		if !column.PrimaryKey() {
			columns = append(columns, column.Name)
			markers = append(markers, d.NextMarker(markerPos))
			values = append(values, column.Value)
		}
	}
	return columns, markers, values
}

func (d *Postgres) Insert(hood *Hood, model *Model) (Id, error) {
	sql, args := d.InsertSql(hood, model)
	var id int64
	err := hood.QueryRow(sql, args...).Scan(&id)
	return Id(id), err
}

func (d *Postgres) InsertSql(hood *Hood, model *Model) (string, []interface{}) {
	m := 0
	columns, markers, values := d.columnsMarkersAndValuesForModel(model, &m)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v) RETURNING %v",
		model.Table,
		strings.Join(columns, ", "),
		strings.Join(markers, ", "),
		model.Pk.Name,
	)
	return sql, values
}

func (d *Postgres) Update(hood *Hood, model *Model) (Id, error) {
	sql, args := d.UpdateSql(hood, model)
	_, err := hood.Exec(sql, args...)
	if err != nil {
		return -1, err
	}
	return model.Pk.Value.(Id), nil
}

func (d *Postgres) UpdateSql(hood *Hood, model *Model) (string, []interface{}) {
	m := 0
	columns, markers, values := d.columnsMarkersAndValuesForModel(model, &m)
	pairs := make([]string, 0, len(columns))
	for i, column := range columns {
		pairs = append(pairs, fmt.Sprintf("%v = %v", column, markers[i]))
	}
	sql := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %v",
		model.Table,
		strings.Join(pairs, ", "),
		model.Pk.Name,
		d.NextMarker(&m),
	)
	values = append(values, model.Pk.Value)
	return sql, values
}

func (d *Postgres) Delete(hood *Hood, model *Model) (Id, error) {
	sql, args := d.DeleteSql(hood, model)
	_, err := hood.Exec(sql, args...)
	return args[0].(Id), err
}

func (d *Postgres) DeleteSql(hood *Hood, model *Model) (string, []interface{}) {
	n := 0
	return fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %v",
		model.Table,
		model.Pk.Name,
		d.NextMarker(&n),
	), []interface{}{model.Pk.Value}
}

func (d *Postgres) CreateTable(hood *Hood, model *Model) error {
	_, err := hood.Exec(d.CreateTableSql(hood, model))
	return err
}

func (d *Postgres) CreateTableSql(hood *Hood, model *Model) string {
	a := []string{"CREATE TABLE ", model.Table, " ( "}
	for i, field := range model.Fields {
		b := []string{
			field.Name,
			d.SqlType(field.Value, field.Size()),
		}
		if field.NotNull() {
			b = append(b, d.KeywordNotNull())
		}
		if x := field.Default(); x != "" {
			b = append(b, d.KeywordDefault(x))
		}
		if field.PrimaryKey() {
			b = append(b, d.KeywordPrimaryKey())
		}
		if incKeyword := d.KeywordAutoIncrement(); field.PrimaryKey() && incKeyword != "" {
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

func (d *Postgres) DropTable(hood *Hood, table string) error {
	_, err := hood.Exec(d.DropTableSql(hood, table))
	return err
}

func (d *Postgres) DropTableSql(hood *Hood, table string) string {
	return fmt.Sprintf("DROP TABLE %v", table)
}

func (d *Postgres) RenameTable(hood *Hood, from, to string) error {
	_, err := hood.Exec(d.RenameTableSql(hood, from, to))
	return err
}

func (d *Postgres) RenameTableSql(hood *Hood, from, to string) string {
	return fmt.Sprintf("ALTER TABLE %v RENAME TO %v", from, to)
}

func (d *Postgres) AddColumn(hood *Hood, table string, column *Field) error {
	_, err := hood.Exec(d.AddColumnSql(hood, table, column))
	return err
}

func (d *Postgres) AddColumnSql(hood *Hood, table string, column *Field) string {
	return fmt.Sprintf(
		"ALTER TABLE %v ADD COLUMN %v %v",
		table,
		column.Name,
		d.SqlType(column.Value, column.Size()),
	)
}

func (d *Postgres) RenameColumn(hood *Hood, table, from, to string) error {
	_, err := hood.Exec(d.RenameColumnSql(hood, table, from, to))
	return err
}

func (d *Postgres) RenameColumnSql(hood *Hood, table, from, to string) string {
	return fmt.Sprintf("ALTER TABLE %v RENAME COLUMN %v TO %v", table, from, to)
}

func (d *Postgres) ChangeColumn(hood *Hood, table string, column *Field) error {
	_, err := hood.Exec(d.ChangeColumnSql(hood, table, column))
	return err
}

func (d *Postgres) ChangeColumnSql(hood *Hood, table string, column *Field) string {
	return fmt.Sprintf("ALTER TABLE %v ALTER COLUMN %v TYPE %v", table, column.Name, d.SqlType(column.Value, column.Size()))
}

func (d *Postgres) KeywordNotNull() string {
	return "NOT NULL"
}

func (d *Postgres) KeywordDefault(s string) string {
	return fmt.Sprintf("DEFAULT %v", s)
}

func (d *Postgres) KeywordPrimaryKey() string {
	return "PRIMARY KEY"
}

func (d *Postgres) KeywordAutoIncrement() string {
	// postgres has not auto increment keyword, uses SERIAL type
	return ""
}
