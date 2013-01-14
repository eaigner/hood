// Package hood provides a database agnostic, transactional ORM for the sql
// package
package hood

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type (
	// Hood is an ORM handle.
	Hood struct {
		Db           *sql.DB
		Dialect      Dialect
		Log          bool
		qo           qo     // the query object
		schema       Schema // keeping track of the schema
		dryRun       bool   // if actual sql is executed or not
		selectCols   []string
		selectTable  string
		whereClauses []string
		whereArgs    []interface{}
		markerPos    int
		limit        int
		offset       int
		orderBy      string
		joinOps      []Join
		joinTables   []interface{}
		joinCol1     []string
		joinCol2     []string
		groupBy      string
		havingCond   string
		havingArgs   []interface{}
	}

	// Id represents a auto-incrementing integer primary key type.
	Id int64

	// Index represents an schema field type for indexes.
	Index int

	// UniqueIndex represents an schema field type for unique indexes.
	UniqueIndex int

	// Varchar represents a VARCHAR type.
	VarChar string

	// Created denotes a timestamp field that is automatically set on insert.
	Created struct {
		time.Time
	}

	// Updated denotes a timestamp field that is automatically set on update.
	Updated struct {
		time.Time
	}

	// Model represents a parsed schema interface{}.
	Model struct {
		Pk      *ModelField
		Table   string
		Fields  []*ModelField
		Indexes []*ModelIndex
	}

	// ModelField represents a schema field of a parsed model.
	ModelField struct {
		Name         string            // Column name
		Value        interface{}       // Value
		SqlTags      map[string]string // The sql struct tags for this field
		ValidateTags map[string]string // The validate struct tags for this field
		RawTag       reflect.StructTag // The raw tag
	}

	// ModelIndex represents a schema index of a parsed model.
	ModelIndex struct {
		Name    string
		Columns []string
		Unique  bool
		RawTag  reflect.StructTag // The raw tag
	}

	// Schema is a collection of models
	Schema []*Model

	// Config represents an environment entry in the config.json file
	Config map[string]string

	// Environment represents a configuration map for each environment specified
	// in the config.json file
	Environment map[string]Config

	qo interface {
		Prepare(query string) (*sql.Stmt, error)
		QueryRow(query string, args ...interface{}) *sql.Row
	}
)

const (
	InnerJoin = Join(iota)
	LeftJoin
	RightJoin
	FullJoin
)

type Join int

// PrimaryKey tests if the field is declared using the sql tag "pk" or is of type Id
func (field *ModelField) PrimaryKey() bool {
	_, isPk := field.SqlTags["pk"]
	_, isId := field.Value.(Id)
	return isPk || isId
}

// NotNull tests if the field is declared as NOT NULL
func (field *ModelField) NotNull() bool {
	_, ok := field.SqlTags["notnull"]
	return ok
}

// Default returns the default value for the field
func (field *ModelField) Default() string {
	return field.SqlTags["default"]
}

// Size returns the field size, e.g. for varchars
func (field *ModelField) Size() int {
	v, ok := field.SqlTags["size"]
	if ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

// Zero tests wether or not the field is set
func (field *ModelField) Zero() bool {
	x := field.Value
	return x == nil || x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

// String returns the field string value and a bool flag indicating if the
// conversion was successful
func (field *ModelField) String() (string, bool) {
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
func (field *ModelField) Int() (int64, bool) {
	switch t := field.Value.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(t).Int(), true
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(t).Uint()), true
	}
	return 0, false
}

func (field *ModelField) GoDeclaration() string {
	t := ""
	if x := field.RawTag; len(x) > 0 {
		t = fmt.Sprintf("\t`%s`", x)
	}
	return fmt.Sprintf(
		"%s\t%s%s",
		snakeToUpperCamel(field.Name),
		reflect.TypeOf(field.Value).String(),
		t,
	)
}

// Validate tests if the field conforms to it's validation constraints specified
// int the "validate" struct tag
func (field *ModelField) Validate() error {
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

func (index *ModelIndex) GoDeclaration() string {
	typ := reflect.TypeOf(Index(0)).String()
	if index.Unique {
		typ = reflect.TypeOf(UniqueIndex(0)).String()
	}
	t := ""
	if x := index.RawTag; len(x) > 0 {
		t = fmt.Sprintf("\t`%s`", index.RawTag)
	}
	return fmt.Sprintf(
		"%s\t%s%s",
		snakeToUpperCamel(index.Name),
		typ,
		t,
	)
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

func (model *Model) GoDeclaration() string {
	a := []string{fmt.Sprintf("type %s struct {", snakeToUpperCamel(model.Table))}
	for _, f := range model.Fields {
		a = append(a, "\t"+f.GoDeclaration())
	}
	if len(model.Indexes) > 0 {
		a = append(a, "", "\t// Indexes")
	}
	for _, i := range model.Indexes {
		a = append(a, "\t"+i.GoDeclaration())
	}
	a = append(a, "}")
	return strings.Join(a, "\n")
}

func (schema Schema) GoDeclaration() string {
	a := []string{}
	for _, m := range schema {
		a = append(a, m.GoDeclaration())
	}
	return strings.Join(a, "\n\n")
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

// Dry creates a new Hood instance for schema generation.
func Dry() *Hood {
	hd := New(nil, nil)
	hd.dryRun = true
	return hd
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

// Load opens a new database from a config.json file with the specified
// environment, or development if none is specified.
func Load(path, env string) (*Hood, error) {
	if env == "" {
		env = "development"
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var envs Environment
	err = dec.Decode(&envs)
	if err != nil {
		return nil, err
	}
	conf, ok := envs[env]
	if !ok {
		return nil, fmt.Errorf("config entry for specified environment '%s' not found", env)
	}
	return Open(conf["driver"], conf["source"])
}

// RegisterDialect registers a new dialect using the specified name and dialect.
func RegisterDialect(name string, dialect Dialect) {
	registeredDialects[name] = dialect
}

// Reset resets the internal state.
func (hood *Hood) Reset() {
	hood.selectCols = nil
	hood.selectTable = ""
	hood.whereClauses = make([]string, 0, 10)
	hood.whereArgs = make([]interface{}, 0, 10)
	hood.markerPos = 0
	hood.limit = 0
	hood.offset = 0
	hood.orderBy = ""
	hood.joinOps = []Join{}
	hood.joinTables = []interface{}{}
	hood.joinCol1 = []string{}
	hood.joinCol2 = []string{}
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

// SchemaDefinition returns a string of the schema represented as Go structs.
func (hood *Hood) SchemaDefinition() string {
	return hood.schema.GoDeclaration()
}

// Select adds a SELECT clause to the query with the specified table and columns.
// The table can either be a string or it's name can be inferred from the passed
// interface{} type.
func (hood *Hood) Select(table interface{}, columns ...string) *Hood {
	hood.selectCols = columns
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
func (hood *Hood) Join(op Join, table2 interface{}, columnt1, columnt2 string) *Hood {
	hood.joinOps = append(hood.joinOps, op)
	hood.joinTables = append(hood.joinTables, table2)
	hood.joinCol1 = append(hood.joinCol1, columnt1)
	hood.joinCol2 = append(hood.joinCol2, columnt2)
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
	if hood.selectTable == "" {
		hood.Select(out)
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
				err = hood.Dialect.SetModelValue(value, field)
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
	result, err := stmt.Exec(hood.convertSpecialTypes(args)...)
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
	return hood.qo.QueryRow(query, hood.convertSpecialTypes(args)...)
}

func (hood *Hood) convertSpecialTypes(a []interface{}) []interface{} {
	args := make([]interface{}, 0, len(a))
	for _, v := range a {
		args = append(args, hood.Dialect.ConvertHoodType(v))
	}
	return args
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
	now := time.Now()
	isUpdate := model.Pk != nil && !model.Pk.Zero()
	if isUpdate {
		err = callModelMethod(f, "BeforeUpdate", false)
		if err != nil {
			return id, err
		}
		for _, f := range model.Fields {
			switch f.Value.(type) {
			case Updated:
				f.Value = now
			}
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
		for _, f := range model.Fields {
			switch f.Value.(type) {
			case Created, Updated:
				f.Value = now
			}
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
			} else if field.Type() == reflect.TypeOf(Updated{}) {
				field.Set(reflect.ValueOf(Updated{now}))
			} else if !isUpdate && field.Type() == reflect.TypeOf(Created{}) {
				field.Set(reflect.ValueOf(Created{now}))
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
	hood.schema = append(hood.schema, model)
	if hood.dryRun {
		return nil
	}
	tx := hood.Begin()
	if ifNotExists {
		tx.Dialect.CreateTableIfNotExists(tx, model)
	} else {
		tx.Dialect.CreateTable(tx, model)
	}
	for _, i := range model.Indexes {
		tx.Dialect.CreateIndex(tx, i.Name, model.Table, i.Unique, i.Columns...)
	}
	return tx.Commit()
}

// DropTable drops the table matching the provided table name.
func (hood *Hood) DropTable(table interface{}) error {
	return hood.dropTable(table, false)
}

// DropTableIfExists drops the table matching the provided table name if it exists.
func (hood *Hood) DropTableIfExists(table interface{}) error {
	return hood.dropTable(table, true)
}

func (hood *Hood) dropTable(table interface{}, ifExists bool) error {
	s := []*Model{}
	for _, m := range hood.schema {
		if m.Table != tableName(table) {
			s = append(s, m)
		}
	}
	hood.schema = s
	if hood.dryRun {
		return nil
	}
	if ifExists {
		return hood.Dialect.DropTableIfExists(hood, tableName(table))
	}
	return hood.Dialect.DropTable(hood, tableName(table))
}

// RenameTable renames a table. The arguments can either be a schema definition
// or plain strings.
func (hood *Hood) RenameTable(from, to interface{}) error {
	for _, m := range hood.schema {
		if m.Table == tableName(from) {
			m.Table = tableName(to)
		}
	}
	if hood.dryRun {
		return nil
	}
	return hood.Dialect.RenameTable(hood, tableName(from), tableName(to))
}

// AddColumns adds the columns in the specified schema to the table.
func (hood *Hood) AddColumns(table, columns interface{}) error {
	m, err := interfaceToModel(columns)
	if err != nil {
		return err
	}
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			if m.Pk != nil {
				panic("primary keys can only be specified on table create (for now)")
			}
			s.Fields = append(s.Fields, m.Fields...)
		}
	}
	if hood.dryRun {
		return nil
	}
	tx := hood.Begin()
	for _, column := range m.Fields {
		err = hood.Dialect.AddColumn(tx, tableName(table), column.Name, column.Value, column.Size())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RenameColumn renames the column in the specified table.
func (hood *Hood) RenameColumn(table interface{}, from, to string) error {
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			for _, f := range s.Fields {
				if f.Name == from {
					f.Name = to
				}
			}
		}
	}
	if hood.dryRun {
		return nil
	}
	return hood.Dialect.RenameColumn(hood, tableName(table), from, to)
}

// ChangeColumn changes the data type of the specified column.
func (hood *Hood) ChangeColumns(table, column interface{}) error {
	m, err := interfaceToModel(column)
	if err != nil {
		return err
	}
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			fields := []*ModelField{}
			for _, oldField := range s.Fields {
				for _, newField := range m.Fields {
					if newField.Name == oldField.Name {
						fields = append(fields, newField)
					} else {
						fields = append(fields, oldField)
					}
				}
			}
			s.Fields = fields
		}
	}
	if hood.dryRun {
		return nil
	}
	tx := hood.Begin()
	for _, column := range m.Fields {
		err = hood.Dialect.ChangeColumn(tx, tableName(table), column.Name, column.Value, column.Size())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (hood *Hood) RemoveColumns(table, columns interface{}) error {
	m, err := interfaceToModel(columns)
	if err != nil {
		return err
	}
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			fields := []*ModelField{}
			for _, field := range s.Fields {
				remove := false
				for _, fieldToRemove := range m.Fields {
					if field.Name == fieldToRemove.Name {
						remove = true
						break
					}
				}
				if !remove {
					fields = append(fields, field)
				}
			}
			s.Fields = fields
		}
	}
	if hood.dryRun {
		return nil
	}
	tx := hood.Begin()
	for _, column := range m.Fields {
		err = hood.Dialect.DropColumn(tx, tableName(table), column.Name)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (hood *Hood) CreateIndex(table, index interface{}) error {
	m, err := interfaceToModel(index)
	if err != nil {
		return err
	}
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			s.Indexes = append(s.Indexes, m.Indexes...)
		}
	}
	if hood.dryRun {
		return nil
	}
	tx := hood.Begin()
	for _, index := range m.Indexes {
		err = hood.Dialect.CreateIndex(tx, index.Name, tableName(table), index.Unique, index.Columns...)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (hood *Hood) DropIndex(name string, table interface{}) error {
	for _, s := range hood.schema {
		if s.Table == tableName(table) {
			indexes := []*ModelIndex{}
			for _, i := range s.Indexes {
				if i.Name != name {
					indexes = append(indexes, i)
				}
			}
			s.Indexes = indexes
		}
	}
	if hood.dryRun {
		return nil
	}
	return hood.Dialect.DropIndex(hood, name)
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

func addIndex(m *Model, field reflect.StructField, unique bool, sqlTags map[string]string, tag reflect.StructTag) {
	if t, ok := sqlTags["columns"]; ok {
		m.Indexes = append(m.Indexes, &ModelIndex{
			Name:    toSnake(field.Name),
			Columns: strings.Split(t, ":"),
			Unique:  unique,
			RawTag:  tag,
		})
	}
}

func addFields(m *Model, t reflect.Type, v reflect.Value) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		sqlTag := field.Tag.Get("sql")
		if sqlTag == "-" {
			continue
		}
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			addFields(m, field.Type, v.Field(i))
			continue
		}
		parsedSqlTags := parseTags(sqlTag)
		if field.Type == reflect.TypeOf(Index(0)) {
			addIndex(m, field, false, parsedSqlTags, field.Tag)
		} else if field.Type == reflect.TypeOf(UniqueIndex(0)) {
			addIndex(m, field, true, parsedSqlTags, field.Tag)
		} else {
			fd := &ModelField{
				Name:         toSnake(field.Name),
				Value:        v.FieldByName(field.Name).Interface(),
				SqlTags:      parsedSqlTags,
				ValidateTags: parseTags(field.Tag.Get("validate")),
				RawTag:       field.Tag,
			}
			if fd.PrimaryKey() {
				m.Pk = fd
			}
			m.Fields = append(m.Fields, fd)
		}
	}
}

func interfaceToModel(f interface{}) (*Model, error) {
	v := reflect.Indirect(reflect.ValueOf(f))
	if v.Kind() != reflect.Struct {
		return nil, errors.New("model is not a struct")
	}
	t := v.Type()
	m := &Model{
		Pk:      nil,
		Table:   interfaceToSnake(f),
		Fields:  []*ModelField{},
		Indexes: []*ModelIndex{},
	}
	addFields(m, t, v)
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
