// Package hood provides a database agnostic, transactional ORM for the sql
// package
package hood

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type (
	// Hood is an ORM handle.
	Hood struct {
		Db        *sql.DB
		Dialect   Dialect
		Log       bool
		qo        qo // the query object
		selector  string
		where     []string
		args      []interface{}
		markerPos int
		limit     string
		offset    string
		orderBy   string
		joins     []string
		groupBy   string
		having    string
	}

	// Id represents a auto-incrementing integer primary key type.
	Id int64

	// Varchar represents a VARCHAR type.
	VarChar string

	// Model represents a parsed schema interface{}.
	Model struct {
		Pk     *Field
		Table  string
		Fields []*Field
	}

	// Field represents a schema field.
	Field struct {
		Name         string            // Column name
		Value        interface{}       // Value
		SqlTags      map[string]string // The sql struct tags for this field
		ValidateTags map[string]string // The validate struct tags for this field
	}
	qo interface {
		Prepare(query string) (*sql.Stmt, error)
		QueryRow(query string, args ...interface{}) *sql.Row
	}
)

// PrimaryKey tests if the field is declared using the sql tag "pk" or is of type Id
func (field *Field) PrimaryKey() bool {
	_, isPk := field.SqlTags["pk"]
	_, isId := field.Value.(Id)
	return isPk || isId
}

// NotNull tests if the field is declared as NOT NULL
func (field *Field) NotNull() bool {
	_, ok := field.SqlTags["notnull"]
	return ok
}

// Default returns the default value for the field
func (field *Field) Default() string {
	return field.SqlTags["default"]
}

// Size returns the field size, e.g. for varchars
func (field *Field) Size() int {
	v, ok := field.SqlTags["size"]
	if ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

// Zero tests wether or not the field is set
func (field *Field) Zero() bool {
	x := field.Value
	return x == nil || x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

// String returns the field string value and a bool flag indicating if the
// conversion was successful
func (field *Field) String() (string, bool) {
	switch t := field.Value.(type) {
	case string:
		return t, true
	case VarChar:
		return string(t), true
	}
	return "", false
}

// Int returns the field int value and a bool flag indication if the conversion
// was successful
func (field *Field) Int() (int64, bool) {
	switch t := field.Value.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(t).Int(), true
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(t).Uint()), true
	}
	return 0, false
}

// Validate tests if the field conforms to it's validation constraints specified
// int the "validate" struct tag
func (field *Field) Validate() error {
	// length
	if tuple, ok := field.ValidateTags["len"]; ok {
		s, ok := field.String()
		if ok {
			if err := validateLen(s, tuple); err != nil {
				return err
			}
		}
	}
	// range
	if tuple, ok := field.ValidateTags["range"]; ok {
		i, ok := field.Int()
		if ok {
			if err := validateRange(i, tuple); err != nil {
				return err
			}
		}
	}
	// presence
	if _, ok := field.ValidateTags["presence"]; ok {
		if field.Zero() {
			return errors.New("value not set")
		}
	}
	return nil
}

func parseTuple(tuple string) (string, string) {
	c := strings.Split(tuple, ":")
	a := c[0]
	b := c[1]
	if len(c) != 2 || (len(a) == 0 && len(b) == 0) {
		panic("invalid validation tuple")
	}
	return a, b
}

func validateLen(s, tuple string) error {
	a, b := parseTuple(tuple)
	if len(a) > 0 {
		min, err := strconv.Atoi(a)
		if err != nil {
			panic(err)
		}
		if len(s) < min {
			return errors.New("value too short")
		}
	}
	if len(b) > 0 {
		max, err := strconv.Atoi(b)
		if err != nil {
			panic(err)
		}
		if len(s) > max {
			return errors.New("value too long")
		}
	}
	return nil
}

func validateRange(i int64, tuple string) error {
	a, b := parseTuple(tuple)
	if len(a) > 0 {
		min, err := strconv.ParseInt(a, 10, 64)
		if err != nil {
			return err
		}
		if i < min {
			return errors.New("value too small")
		}
	}
	if len(b) > 0 {
		max, err := strconv.ParseInt(b, 10, 64)
		if err != nil {
			return err
		}
		if i > max {
			return errors.New("value too big")
		}
	}
	return nil
}

func (model *Model) Validate() error {
	for _, field := range model.Fields {
		err := field.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

var registeredDialects map[string]Dialect = make(map[string]Dialect)

// New creates a new Hood using the specified DB and dialect.
func New(database *sql.DB, dialect Dialect) *Hood {
	hood := &Hood{
		Db:      database,
		Dialect: dialect,
		qo:      database,
	}
	hood.Reset()

	return hood
}

// Open opens a new database connection using the specified driver and data
// source name. It matches the sql.Open method signature.
func Open(driverName, dataSourceName string) (*Hood, error) {
	database, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	dialect := registeredDialects[driverName]
	if dialect == nil {
		return nil, errors.New("no dialect registered for driver name")
	}
	return New(database, dialect), nil
}

// RegisterDialect registers a new dialect using the specified name and dialect.
func RegisterDialect(name string, dialect Dialect) {
	registeredDialects[name] = dialect
}

// Reset resets the internal state.
func (hood *Hood) Reset() {
	hood.selector = ""
	hood.where = make([]string, 0, 10)
	hood.args = []interface{}{}
	hood.markerPos = 0
	hood.limit = ""
	hood.offset = ""
	hood.orderBy = ""
	hood.joins = []string{}
	hood.groupBy = ""
	hood.having = ""
}

// Begin starts a new transaction and returns a copy of Hood. You have to call
// subsequent methods on the newly returned object.
func (hood *Hood) Begin() *Hood {
	c := new(Hood)
	*c = *hood
	q, err := hood.Db.Begin()
	if err != nil {
		panic(err)
	}
	c.qo = q

	return c
}

// Commit commits a started transaction.
func (hood *Hood) Commit() error {
	if v, ok := hood.qo.(*sql.Tx); ok {
		return v.Commit()
	}
	return nil
}

// Rollback rolls back a started transaction.
func (hood *Hood) Rollback() error {
	if v, ok := hood.qo.(*sql.Tx); ok {
		return v.Rollback()
	}
	return nil
}

// Select adds a SELECT clause to the query with the specified columsn and table.
// The table can either be a string or it's name can be inferred from the passed
// interface{} type.
func (hood *Hood) Select(selector string, table interface{}) *Hood {
	if selector == "" {
		selector = "*"
	}
	from := ""
	switch f := table.(type) {
	case string:
		from = f
	case interface{}:
		from = interfaceToSnake(f)
	}
	if from == "" {
		panic("FROM cannot be empty")
	}
	hood.selector = fmt.Sprintf("SELECT %v FROM %v", selector, from)

	return hood
}

// Where adds a WHERE clause to the query. The markers are database agnostic, so
// you can always use ? and it will get replaced with the dialect specific
// version, for example
//   Where("id = ?", 3)
func (hood *Hood) Where(query string, args ...interface{}) *Hood {
	hood.where = append(hood.where, query)
	hood.args = append(hood.args, args...)

	return hood
}

// Limit adds a LIMIT clause to the query.
func (hood *Hood) Limit(limit int) *Hood {
	hood.limit = "LIMIT ?"
	hood.args = append(hood.args, limit)
	return hood
}

// Offset adds an OFFSET clause to the query.
func (hood *Hood) Offset(offset int) *Hood {
	hood.offset = "OFFSET ?"
	hood.args = append(hood.args, offset)
	return hood
}

// OrderBy adds an ORDER BY clause to the query.
func (hood *Hood) OrderBy(key string) *Hood {
	hood.orderBy = fmt.Sprintf("ORDER BY %v", key)
	return hood
}

// Join performs a JOIN on tables, for example
//   Join("INNER JOIN", "users", "orders.user_id == users.id")
func (hood *Hood) Join(op, table, condition string) *Hood {
	hood.joins = append(hood.joins, fmt.Sprintf("%v JOIN %v ON %v", op, table, condition))
	return hood
}

// GroupBy adds a GROUP BY clause to the query.
func (hood *Hood) GroupBy(key string) *Hood {
	hood.groupBy = fmt.Sprintf("GROUP BY %v", key)
	return hood
}

// Having adds a HAVING clause to the query.
func (hood *Hood) Having(condition string, args ...interface{}) *Hood {
	hood.having = fmt.Sprintf("HAVING %v", condition)
	hood.args = append(hood.args, args...)
	return hood
}

// Find performs a find using the previously specified query. If no explicit
// SELECT clause was specified earlier, the select is inferred from the passed
// interface type.
func (hood *Hood) Find(out interface{}) error {
	defer hood.Reset()
	panicMsg := errors.New("expected pointer to struct slice *[]struct")
	if x := reflect.TypeOf(out).Kind(); x != reflect.Ptr {
		panic(panicMsg)
	}
	sliceValue := reflect.Indirect(reflect.ValueOf(out))
	if x := sliceValue.Kind(); x != reflect.Slice {
		panic(panicMsg)
	}
	sliceType := sliceValue.Type().Elem()
	if x := sliceType.Kind(); x != reflect.Struct {
		panic(panicMsg)
	}
	// infer the select statement from the type if not set
	if hood.selector == "" {
		hood.Select("*", out)
	}
	query := hood.querySql()
	if hood.Log {
		fmt.Println(query)
	}
	stmt, err := hood.qo.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if hood.Log {
		fmt.Println(hood.args)
	}
	rows, err := stmt.Query(hood.args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	for rows.Next() {
		containers := make([]interface{}, 0, len(cols))
		for i := 0; i < cap(containers); i++ {
			var v interface{}
			containers = append(containers, &v)
		}
		err := rows.Scan(containers...)
		if err != nil {
			return err
		}
		// create a new row and fill
		rowValue := reflect.New(sliceType)
		for i, v := range containers {
			key := cols[i]
			value := reflect.Indirect(reflect.ValueOf(v))
			name := snakeToUpperCamel(key)
			field := rowValue.Elem().FieldByName(name)
			if field.IsValid() {
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
			}
		}
		// append to output
		sliceValue.Set(reflect.Append(sliceValue, rowValue.Elem()))
	}
	return nil
}

// Exec executes a raw sql query.
func (hood *Hood) Exec(query string, args ...interface{}) (sql.Result, error) {
	defer hood.Reset()
	query = hood.substituteMarkers(query)
	if hood.Log {
		fmt.Println(query)
	}
	stmt, err := hood.qo.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	if hood.Log {
		fmt.Println(args...)
	}
	result, err := stmt.Exec(convertSpecialTypes(args)...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always return a non-nil value. Errors are deferred until Row's Scan
// method is called.
func (hood *Hood) QueryRow(query string, args ...interface{}) *sql.Row {
	if hood.Log {
		fmt.Println(query)
		fmt.Println(args...)
	}
	return hood.qo.QueryRow(query, convertSpecialTypes(args)...)
}

// Validate validates the provided struct
func (hood *Hood) Validate(f interface{}) error {
	model, err := interfaceToModel(f)
	if err != nil {
		return err
	}
	err = model.Validate()
	if err != nil {
		return err
	}
	return nil
}

// Save performs an INSERT, or UPDATE if the passed structs Id is set.
func (hood *Hood) Save(f interface{}) (Id, error) {
	var (
		id  Id
		err error
	)
	model, err := interfaceToModel(f)
	if err != nil {
		return -1, err
	}
	err = model.Validate()
	if err != nil {
		return -1, err
	}
	if model.Pk != nil && !model.Pk.Zero() {
		id, err = hood.update(model)
	} else {
		id, err = hood.insert(model)
	}
	if err != nil {
		return -1, err
	}
	// update model id after save
	structValue := reflect.Indirect(reflect.ValueOf(f))
	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		if field.Type() == reflect.TypeOf(Id(0)) {
			field.SetInt(int64(id))
		}
	}
	return id, err
}

func (hood *Hood) doAll(f interface{}, doFunc func(f2 interface{}) (Id, error)) ([]Id, error) {
	panicMsg := "expected pointer to struct slice *[]struct"
	if reflect.TypeOf(f).Kind() != reflect.Ptr {
		panic(panicMsg)
	}
	if reflect.TypeOf(f).Elem().Kind() != reflect.Slice {
		panic(panicMsg)
	}
	sliceValue := reflect.ValueOf(f).Elem()
	sliceLen := sliceValue.Len()
	ids := make([]Id, 0, sliceLen)
	for i := 0; i < sliceLen; i++ {
		id, err := doFunc(sliceValue.Index(i).Addr().Interface())
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SaveAll performs an INSERT or UPDATE on a slice of structs.
func (hood *Hood) SaveAll(f interface{}) ([]Id, error) {
	return hood.doAll(f, func(f2 interface{}) (Id, error) {
		return hood.Save(f2)
	})
}

// Delete deletes the row matching the specified structs Id.
func (hood *Hood) Delete(f interface{}) (Id, error) {
	model, err := interfaceToModel(f)
	if err != nil {
		return -1, err
	}
	id, err := hood.delete(model)
	if err != nil {
		return -1, err
	}
	return id, err
}

// DeleteAll deletes the rows matching the specified struct slice Ids.
func (hood *Hood) DeleteAll(f interface{}) ([]Id, error) {
	return hood.doAll(f, func(f2 interface{}) (Id, error) {
		return hood.Delete(f2)
	})
}

// CreateTable creates a new table based on the provided schema.
func (hood *Hood) CreateTable(table interface{}) error {
	model, err := interfaceToModel(table)
	if err != nil {
		return err
	}
	_, err = hood.Exec(hood.createTableSql(model))
	if err != nil {
		return err
	}
	return nil
}

// DropTable drops the table matching the provided table name.
func (hood *Hood) DropTable(table interface{}) error {
	model, err := interfaceToModel(table)
	if err != nil {
		return err
	}
	_, err = hood.Exec(fmt.Sprintf("DROP TABLE %v", model.Table))
	if err != nil {
		return err
	}
	return nil
}

func (hood *Hood) insert(model *Model) (Id, error) {
	if model.Pk == nil {
		panic("no primary key field")
	}
	query, values := hood.insertSql(model)
	// check if dialect requires custom insert (like PostgreSQL)
	if id, err, ok := hood.Dialect.Insert(hood, model, query, values...); ok {
		return id, err
	}
	result, err := hood.Exec(query, values...)
	if err != nil {
		return -1, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return Id(id), nil
}

func (hood *Hood) insertSql(model *Model) (string, []interface{}) {
	defer hood.Reset()
	keys, values, markers := hood.keysValuesAndMarkersForModel(model)
	stmt := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v)",
		model.Table,
		strings.Join(keys, ", "),
		strings.Join(markers, ", "),
	)
	return stmt, values
}

func (hood *Hood) update(model *Model) (Id, error) {
	if model.Pk == nil {
		panic("no primary key field")
	}
	query, values := hood.updateSql(model)
	_, err := hood.Exec(query, values...)
	if err != nil {
		return -1, err
	}
	return model.Pk.Value.(Id), nil
}

func (hood *Hood) updateSql(model *Model) (string, []interface{}) {
	defer hood.Reset()
	keys, values, markers := hood.keysValuesAndMarkersForModel(model)
	pairs := make([]string, 0, len(keys))
	for i, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%v = %v", key, markers[i]))
	}
	stmt := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %v",
		model.Table,
		strings.Join(pairs, ", "),
		model.Pk.Name,
		hood.nextMarker(),
	)
	return stmt, append(values, model.Pk.Value)
}

func (hood *Hood) delete(model *Model) (Id, error) {
	if model.Pk == nil {
		panic("no primary key field")
	}
	query, values := hood.deleteSql(model)
	_, err := hood.Exec(query, values...)
	if err != nil {
		return -1, err
	}
	return model.Pk.Value.(Id), nil
}

func (hood *Hood) deleteSql(model *Model) (string, []interface{}) {
	defer hood.Reset()
	stmt := fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %v",
		model.Table,
		model.Pk.Name,
		hood.nextMarker(),
	)
	return stmt, []interface{}{model.Pk.Value}
}

func (hood *Hood) querySql() string {
	query := make([]string, 0, 20)
	if hood.selector != "" {
		query = append(query, hood.selector)
	}
	for _, join := range hood.joins {
		query = append(query, join)
	}
	if x := hood.where; len(x) > 0 {
		query = append(query, fmt.Sprintf("WHERE %v", strings.Join(x, " AND ")))
	}
	if x := hood.groupBy; x != "" {
		query = append(query, x)
	}
	if x := hood.having; x != "" {
		query = append(query, x)
	}
	if x := hood.orderBy; x != "" {
		query = append(query, x)
	}
	if x := hood.limit; x != "" {
		query = append(query, x)
	}
	if x := hood.offset; x != "" {
		query = append(query, x)
	}
	return hood.substituteMarkers(strings.Join(query, " "))
}

func (hood *Hood) createTableSql(model *Model) string {
	a := []string{"CREATE TABLE ", model.Table, " ( "}
	for i, field := range model.Fields {
		b := []string{
			field.Name,
			hood.Dialect.SqlType(field.Value, field.Size()),
		}
		if incStmt := hood.Dialect.StmtAutoIncrement(); field.PrimaryKey() && incStmt != "" {
			b = append(b, incStmt)
		}
		if field.NotNull() {
			b = append(b, hood.Dialect.StmtNotNull())
		}
		if x := field.Default(); x != "" {
			b = append(b, hood.Dialect.StmtDefault(x))
		}
		if field.PrimaryKey() {
			b = append(b, hood.Dialect.StmtPrimaryKey())
		}
		a = append(a, strings.Join(b, " "))
		if i < len(model.Fields)-1 {
			a = append(a, ", ")
		}
	}
	a = append(a, " )")

	return strings.Join(a, "")
}

func (hood *Hood) keysValuesAndMarkersForModel(model *Model) ([]string, []interface{}, []string) {
	max := len(model.Fields)
	keys := make([]string, 0, max)
	values := make([]interface{}, 0, max)
	markers := make([]string, 0, max)
	for _, field := range model.Fields {
		if !field.PrimaryKey() {
			keys = append(keys, field.Name)
			markers = append(markers, hood.nextMarker())
			values = append(values, field.Value)
		}
	}
	return keys, values, markers
}

func (hood *Hood) substituteMarkers(query string) string {
	// in order to use a uniform marker syntax, substitute
	// all question marks with the dialect marker
	chunks := make([]string, 0, len(query)*2)
	for _, v := range query {
		if v == '?' {
			chunks = append(chunks, hood.nextMarker())
		} else {
			chunks = append(chunks, string(v))
		}
	}
	return strings.Join(chunks, "")
}

func (hood *Hood) nextMarker() string {
	marker := hood.Dialect.Marker(hood.markerPos)
	hood.markerPos++
	return marker
}

func parseTags(s string) map[string]string {
	c := strings.Split(s, ",")
	m := make(map[string]string)
	for _, v := range c {
		c2 := strings.Split(v, "(")
		if len(c2) == 2 && len(c2[1]) > 1 {
			m[c2[0]] = c2[1][:len(c2[1])-1]
		} else {
			m[v] = ""
		}
	}
	return m
}

func interfaceToModel(f interface{}) (*Model, error) {
	v := reflect.Indirect(reflect.ValueOf(f))
	if v.Kind() != reflect.Struct {
		return nil, errors.New("model is not a struct")
	}
	t := v.Type()
	m := &Model{
		Pk:     nil,
		Table:  interfaceToSnake(f),
		Fields: []*Field{},
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fd := &Field{
			Name:         toSnake(field.Name),
			Value:        v.FieldByName(field.Name).Interface(),
			SqlTags:      parseTags(field.Tag.Get("sql")),
			ValidateTags: parseTags(field.Tag.Get("validate")),
		}
		if fd.PrimaryKey() {
			m.Pk = fd
		}
		m.Fields = append(m.Fields, fd)
	}
	return m, nil
}

func convertSpecialTypes(a []interface{}) []interface{} {
	args := make([]interface{}, 0, len(a))
	for _, v := range a {
		args = append(args, convertSpecialType(v))
	}
	return args
}

func convertSpecialType(f interface{}) interface{} {
	switch t := f.(type) {
	case VarChar:
		return string(t)
	}
	return f
}
