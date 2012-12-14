package hood

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type Hood struct {
	Db         *sql.DB
	Dialect    Dialect
	Qo         Qo // the query object
	where      string
	params     []interface{}
	paramNum   int
	limit      int
	offset     int
	orderBy    string
	selectCols string
	joins      []string
	groupBy    string
	having     string
}

type Qo interface {
	Prepare(query string) (*sql.Stmt, error)
}

type (
	Id int64
	Pk struct {
		Name string
		Type reflect.Type
	}
	Model map[string]interface{}
)

func New(database *sql.DB, dialect Dialect) *Hood {
	return &Hood{
		Db:      database,
		Dialect: dialect,
	}
}

func (hood *Hood) Begin() *Hood {
	c := new(Hood)
	*c = *hood
	q, err := hood.Db.Begin()
	if err != nil {
		panic(err)
	}
	c.Qo = q

	return c
}

func (hood *Hood) Commit() error {
	if v, ok := hood.Qo.(*sql.Tx); ok {
		return v.Commit()
	}
	return nil
}

func (hood *Hood) Where(query interface{}, args ...interface{}) *Hood {
	switch typedQuery := query.(type) {
	case string: // custom query
		hood.where = hood.substituteMarkers(typedQuery)
		hood.params = args
	case int: // id provided
		hood.where = fmt.Sprintf(
			"%v%v%v = %v",
			hood.Dialect.Quote(),
			hood.Dialect.Pk(),
			hood.Dialect.Quote(),
			hood.nextMarker(),
		)
		hood.params = []interface{}{typedQuery}
	}
	return hood
}

func (hood *Hood) Limit(limit int) *Hood {
	hood.limit = limit
	return hood
}

func (hood *Hood) Offset(offset int) *Hood {
	hood.offset = offset
	return hood
}

func (hood *Hood) OrderBy(order string) *Hood {
	hood.orderBy = order
	return hood
}

func (hood *Hood) Select(columns string) *Hood {
	hood.selectCols = columns
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

func (hood *Hood) Having(condition string) *Hood {
	hood.having = fmt.Sprintf("HAVING %v", condition)
	return hood
}

func (hood *Hood) Find(out interface{}) error {
	// TODO: implement
	return nil
}

func (hood *Hood) FindAll(out []interface{}) error {
	// TODO: implement
	return nil
}

func (hood *Hood) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (hood *Hood) Save(model interface{}) (Id, error) {
	ids, err := hood.SaveAll([]interface{}{model})
	if err != nil {
		return -1, err
	}
	return ids[0], err
}

func (hood *Hood) SaveAll(model []interface{}) ([]Id, error) {
	// TODO: implement
	return nil, nil
}

func (hood *Hood) Destroy(model interface{}) (Id, error) {
	ids, err := hood.DestroyAll([]interface{}{model})
	if err != nil {
		return -1, err
	}
	return ids[0], err
}

func (hood *Hood) DestroyAll(model []interface{}) ([]Id, error) {
	// TODO: implement
	return nil, nil
}

func (hood *Hood) substituteMarkers(query string) string {
	// in order to use a uniform marker syntax, substitute
	// all question marks with the dialect marker
	chunks := []string{}
	for _, v := range strings.Split(query, "?") {
		hood.paramNum++
		chunks = append(chunks, v, hood.nextMarker())
	}
	return strings.Join(chunks, "")
}

func (hood *Hood) nextMarker() string {
	hood.paramNum++
	return hood.Dialect.Marker(hood.paramNum)
}
