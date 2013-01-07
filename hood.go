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
)

type (
	// Hood is an ORM handle.
	Hood struct {
		Db           *sql.DB
		Dialect      Dialect
		Log          bool
		qo           qo // the query object
		selectCols   string
		selectTable  string
		whereClauses []string
		whereArgs    []interface{}
		markerPos    int
		limit        int
		offset       int
		orderBy      string
		joinOps      []string
		joinTables   []string
		joinCond     []string
		groupBy      string
		havingCond   string
		havingArgs   []interface{}
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
			return NewValidationError(ValidationErrorValueNotSet, "value not set")
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
			return NewValidationError(ValidationErrorValueTooShort, "value too short")
		}
	}
	if len(b) > 0 {
		max, err := strconv.Atoi(b)
		if err != nil {
			panic(err)
		}
		if len(s) > max {
			return NewValidationError(ValidationErrorValueTooLong, "value too long")
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
			return NewValidationError(ValidationErrorValueTooSmall, "value too small")
		}
	}
	if len(b) > 0 {
		max, err := strconv.ParseInt(b, 10, 64)
		if err != nil {
			return err
		}
		if i > max {
			return NewValidationError(ValidationErrorValueTooBig, "value too big")
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
	hood.selectCols = ""
	hood.selectTable = ""
	hood.whereClauses = make([]string, 0, 10)
	hood.whereArgs = make([]interface{}, 0, 10)
	hood.markerPos = 0
	hood.limit = 0
	hood.offset = 0
	hood.orderBy = ""
	hood.joinOps = []string{}
	hood.joinTables = []string{}
	hood.joinCond = []string{}
	hood.groupBy = ""
	hood.havingCond = ""
	hood.havingArgs = make([]interface{}, 0, 20)
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
	hood.selectCols = selector
	switch f := table.(type) {
	case string:
		hood.selectTable = f
	case interface{}:
		hood.selectTable = interfaceToSnake(f)
	default:
		panic("invalid table")
	}
	return hood
}

// Where adds a WHERE clause to the query. The markers are database agnostic, so
// you can always use ? and it will get replaced with the dialect specific
// version, for example
//   Where("id = ?", 3)
func (hood *Hood) Where(query string, args ...interface{}) *Hood {
	hood.whereClauses = append(hood.whereClauses, query)
	hood.whereArgs = append(hood.whereArgs, args...)
	return hood
}

// Limit adds a LIMIT clause to the query.
func (hood *Hood) Limit(limit int) *Hood {
	hood.limit = limit
	return hood
}

// Offset adds an OFFSET clause to the query.
func (hood *Hood) Offset(offset int) *Hood {
	hood.offset = offset
	return hood
}

// OrderBy adds an ORDER BY clause to the query.
func (hood *Hood) OrderBy(key string) *Hood {
	hood.orderBy = key
	return hood
}

// Join performs a JOIN on tables, for example
//   Join("INNER JOIN", "users", "orders.user_id == users.id")
func (hood *Hood) Join(op, table, condition string) *Hood {
	hood.joinOps = append(hood.joinOps, op)
	hood.joinTables = append(hood.joinTables, table)
	hood.joinCond = append(hood.joinCond, condition)
	return hood
}

// GroupBy adds a GROUP BY clause to the query.
func (hood *Hood) GroupBy(key string) *Hood {
	hood.groupBy = key
	return hood
}

// Having adds a HAVING clause to the query.
func (hood *Hood) Having(condition string, args ...interface{}) *Hood {
	hood.havingCond = condition
	hood.havingArgs = append(hood.havingArgs, args...)
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
	if hood.selectCols == "" {
		hood.Select("*", out)
	}
	query, args := hood.Dialect.QuerySql(hood)
	if hood.Log {
		fmt.Println(query)
		fmt.Println(args)
	}
	stmt, err := hood.qo.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	rows, err := stmt.Query(args...)
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
				err = hood.Dialect.ValueToField(value, field)
				if err != nil {
					return err
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
	// TODO(erik): model after .Exec(...)
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
	// call validate methods
	err = callModelMethod(f, "Validate", true)
	if err != nil {
		return err
	}
	return nil
}

func callModelMethod(f interface{}, methodName string, isPrefix bool) error {
	typ := reflect.TypeOf(f)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if (isPrefix && strings.HasPrefix(method.Name, methodName)) ||
			(!isPrefix && method.Name == methodName) {
			ft := method.Func.Type()
			if ft.NumOut() == 1 &&
				ft.NumIn() == 1 {
				v := reflect.ValueOf(f).Method(i).Call([]reflect.Value{})
				if vdErr, ok := v[0].Interface().(error); ok {
					return vdErr
				}
			}
		}
	}
	return nil
}

// Save performs an INSERT, or UPDATE if the passed structs Id is set.
func (hood *Hood) Save(f interface{}) (Id, error) {
	var (
		id  Id = -1
		err error
	)
	model, err := interfaceToModel(f)
	if err != nil {
		return id, err
	}
	err = model.Validate()
	if err != nil {
		return id, err
	}
	err = callModelMethod(f, "BeforeSave", false)
	if err != nil {
		return id, err
	}
	if model.Pk == nil {
		panic("no primary key field")
	}
	if model.Pk != nil && !model.Pk.Zero() {
		err = callModelMethod(f, "BeforeUpdate", false)
		if err != nil {
			return id, err
		}
		id, err = hood.Dialect.Update(hood, model)
		if err == nil {
			err = callModelMethod(f, "AfterUpdate", false)
		}
	} else {
		err = callModelMethod(f, "BeforeInsert", false)
		if err != nil {
			return id, err
		}
		id, err = hood.Dialect.Insert(hood, model)
		if err == nil {
			err = callModelMethod(f, "AfterInsert", false)
		}
	}
	if err == nil {
		err = callModelMethod(f, "AfterSave", false)
	}
	if id != -1 {
		// update model id after save
		structValue := reflect.Indirect(reflect.ValueOf(f))
		for i := 0; i < structValue.NumField(); i++ {
			field := structValue.Field(i)
			if field.Type() == reflect.TypeOf(Id(0)) {
				field.SetInt(int64(id))
			}
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
	err = callModelMethod(f, "BeforeDelete", false)
	if err != nil {
		return -1, err
	}
	if model.Pk == nil {
		panic("no primary key field")
	}
	id, err := hood.Dialect.Delete(hood, model)
	if err != nil {
		return -1, err
	}
	return id, callModelMethod(f, "AfterDelete", false)
}

// DeleteAll deletes the rows matching the specified struct slice Ids.
func (hood *Hood) DeleteAll(f interface{}) ([]Id, error) {
	return hood.doAll(f, func(f2 interface{}) (Id, error) {
		return hood.Delete(f2)
	})
}

// CreateTable creates a new table based on the provided schema.
func (hood *Hood) CreateTable(table interface{}) error {
	return hood.createTable(table, false)
}

// CreateTableIfNotExists creates a new table based on the provided schema
// if it does not exist yet.
func (hood *Hood) CreateTableIfNotExists(table interface{}) error {
	return hood.createTable(table, true)
}

func (hood *Hood) createTable(table interface{}, ifNotExists bool) error {
	model, err := interfaceToModel(table)
	if err != nil {
		return err
	}
	if ifNotExists {
		return hood.Dialect.CreateTableIfNotExists(hood, model)
	}
	return hood.Dialect.CreateTable(hood, model)
}

// DropTable drops the table matching the provided table name.
func (hood *Hood) DropTable(table interface{}) error {
	return hood.Dialect.DropTable(hood, tableName(table))
}

// DropTableIfExists drops the table matching the provided table name if it exists.
func (hood *Hood) DropTableIfExists(table interface{}) error {
	return hood.Dialect.DropTableIfExists(hood, tableName(table))
}

// RenameTable renames a table. The arguments can either be a schema definition
// or plain strings.
func (hood *Hood) RenameTable(from, to interface{}) error {
	return hood.Dialect.RenameTable(hood, tableName(from), tableName(to))
}

// AddColumns adds the columns in the specified schema to the table.
func (hood *Hood) AddColumns(schema interface{}) error {
	m, err := interfaceToModel(schema)
	if err != nil {
		return err
	}
	tx := hood.Begin()
	for _, column := range m.Fields {
		err = hood.Dialect.AddColumn(tx, m.Table, column.Name, column.Value, column.Size())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RenameColumn renames the column in the specified table.
func (hood *Hood) RenameColumn(table interface{}, from, to string) error {
	return hood.Dialect.RenameColumn(hood, tableName(table), from, to)
}

// ChangeColumn changes the data type of the specified column.
func (hood *Hood) ChangeColumn(table, column interface{}) error {
	m, err := interfaceToModel(column)
	if err != nil {
		return err
	}
	field := m.Fields[0]
	return hood.Dialect.ChangeColumn(hood, tableName(table), field.Name, field.Value, field.Size())
}

func (hood *Hood) RemoveColumn(table, column interface{}) error {
	m, err := interfaceToModel(column)
	if err != nil {
		return err
	}
	field := m.Fields[0]
	return hood.Dialect.DropColumn(hood, tableName(table), field.Name)
}

func (hood *Hood) CreateIndex(table interface{}, column string, unique bool) error {
	return hood.Dialect.CreateIndex(hood, tableName(table), column, unique)
}

func (hood *Hood) DropIndex(column string) error {
	return hood.Dialect.DropIndex(hood, column)
}

func (hood *Hood) substituteMarkers(query string) string {
	// in order to use a uniform marker syntax, substitute
	// all question marks with the dialect marker
	chunks := make([]string, 0, len(query)*2)
	for _, v := range query {
		if v == '?' {
			chunks = append(chunks, hood.Dialect.NextMarker(&hood.markerPos))
		} else {
			chunks = append(chunks, string(v))
		}
	}
	return strings.Join(chunks, "")
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

func tableName(f interface{}) string {
	switch t := f.(type) {
	case string:
		return t
	}
	m, _ := interfaceToModel(f)
	if m != nil {
		return m.Table
	}
	panic("invalid table name")
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
