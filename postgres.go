package hood

import (
	"fmt"
	_ "github.com/bmizerany/pq"
	"strings"
	"time"
)

func init() {
	RegisterDialect("postgres", &Postgres{})
}

type Postgres struct {
	Base
}

func NewPostgres() Dialect {
	d := &Postgres{}
	d.Base.Dialect = d
	return d
}

func (d *Base) SqlType(f interface{}, size int) string {
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

func (d *Postgres) Insert(hood *Hood, model *Model) (Id, error) {
	sql, args := d.Dialect.InsertSql(hood, model)
	var id int64
	err := hood.QueryRow(sql, args...).Scan(&id)
	return Id(id), err
}

func (d *Postgres) InsertSql(hood *Hood, model *Model) (string, []interface{}) {
	m := 0
	columns, markers, values := columnsMarkersAndValuesForModel(d.Dialect, model, &m)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v) RETURNING %v",
		model.Table,
		strings.Join(columns, ", "),
		strings.Join(markers, ", "),
		model.Pk.Name,
	)
	return sql, values
}

func (d *Postgres) KeywordAutoIncrement() string {
	// postgres has not auto increment keyword, uses SERIAL type
	return ""
}
