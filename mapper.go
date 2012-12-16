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
	Hood struct {
		Db        *sql.DB
		Dialect   Dialect
		Log       bool
		qo        qo // the query object
		selector  string
		where     string
		args      []interface{}
		markerPos int
		limit     string
		offset    string
		orderBy   string
		joins     []string
		groupBy   string
		having    string
	}
	Id      int64
	VarChar string
	Model   struct {
		Pk     *Field
		Table  string
		Fields []*Field
	}
	Field struct {
		Name    string      // column name
		Value   interface{} // value
		IsPk    bool        // primary key
		NotNull bool        // null allowed
		Default string      // default value
		Size    int         // size (e.g. for varchar)
	}
	qo interface {
		Prepare(query string) (*sql.Stmt, error)
		QueryRow(query string, args ...interface{}) *sql.Row
	}
)

var registeredDialects map[string]Dialect = make(map[string]Dialect)

func New(database *sql.DB, dialect Dialect) *Hood {
	hood := &Hood{
		Db:      database,
		Dialect: dialect,
		qo:      database,
	}
	hood.Reset()

	return hood
}

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

func RegisterDialect(name string, dialect Dialect) {
	registeredDialects[name] = dialect
}

func (hood *Hood) Reset() {
	hood.selector = ""
	hood.where = ""
	hood.args = []interface{}{}
	hood.markerPos = 0
	hood.limit = ""
	hood.offset = ""
	hood.orderBy = ""
	hood.joins = []string{}
	hood.groupBy = ""
	hood.having = ""
}

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

func (hood *Hood) Commit() error {
	if v, ok := hood.qo.(*sql.Tx); ok {
		return v.Commit()
	}
	return nil
}

func (hood *Hood) Rollback() error {
	if v, ok := hood.qo.(*sql.Tx); ok {
		return v.Rollback()
	}
	return nil
}

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

func (hood *Hood) Where(query string, args ...interface{}) *Hood {
	hood.where = fmt.Sprintf("WHERE %v", query)
	hood.args = append(hood.args, args...)

	return hood
}

func (hood *Hood) Limit(limit int) *Hood {
	hood.limit = "LIMIT ?"
	hood.args = append(hood.args, limit)
	return hood
}

func (hood *Hood) Offset(offset int) *Hood {
	hood.offset = "OFFSET ?"
	hood.args = append(hood.args, offset)
	return hood
}

func (hood *Hood) OrderBy(key string) *Hood {
	hood.orderBy = fmt.Sprintf("ORDER BY %v", key)
	return hood
}

func (hood *Hood) Join(op, table, condition string) *Hood {
	hood.joins = append(hood.joins, fmt.Sprintf("%v JOIN %v ON %v", op, table, condition))
	return hood
}

func (hood *Hood) GroupBy(key string) *Hood {
	hood.groupBy = fmt.Sprintf("GROUP BY %v", key)
	return hood
}

func (hood *Hood) Having(condition string, args ...interface{}) *Hood {
	hood.having = fmt.Sprintf("HAVING %v", condition)
	hood.args = append(hood.args, args...)
	return hood
}

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

func (hood *Hood) QueryRow(query string, args ...interface{}) *sql.Row {
	if hood.Log {
		fmt.Println(query)
		fmt.Println(args...)
	}
	return hood.qo.QueryRow(query, convertSpecialTypes(args)...)
}

func (hood *Hood) Save(f interface{}) (Id, error) {
	var (
		id  Id
		err error
	)
	model, err := interfaceToModel(f)
	if err != nil {
		return -1, err
	}
	update := false
	if model.Pk != nil {
		if v, ok := model.Pk.Value.(Id); ok && v > 0 {
			update = true
		}
	}
	if update {
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

func (hood *Hood) SaveAll(f interface{}) ([]Id, error) {
	return hood.doAll(f, func(f2 interface{}) (Id, error) {
		return hood.Save(f2)
	})
}

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

func (hood *Hood) DeleteAll(f interface{}) ([]Id, error) {
	return hood.doAll(f, func(f2 interface{}) (Id, error) {
		return hood.Delete(f2)
	})
}

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
	if x := hood.where; x != "" {
		query = append(query, x)
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
			hood.Dialect.SqlType(field.Value, field.Size),
		}
		if incStmt := hood.Dialect.StmtAutoIncrement(); field.IsPk && incStmt != "" {
			b = append(b, incStmt)
		}
		if field.NotNull {
			b = append(b, hood.Dialect.StmtNotNull())
		}
		if field.Default != "" {
			b = append(b, hood.Dialect.StmtDefault(field.Default))
		}
		if field.IsPk {
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
		if !field.IsPk {
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
		tags := parseTags(field.Tag.Get("sql"))
		_, isPk := tags["pk"]
		isPk = isPk || field.Type == reflect.TypeOf(Id(0))
		_, notNull := tags["notnull"]
		defVal, defOk := tags["default"]
		sizeVal, sizeOk := tags["size"]
		fd := &Field{
			Name:    toSnake(field.Name),
			Value:   v.FieldByName(field.Name).Interface(),
			IsPk:    isPk,
			NotNull: notNull,
		}
		// set default value
		if defOk {
			fd.Default = defVal
		}
		// set size
		if sizeOk {
			intVal, _ := strconv.Atoi(sizeVal)
			fd.Size = intVal
		}
		// if it's a primary key, set reference
		if isPk {
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
