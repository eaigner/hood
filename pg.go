package hood

import (
	"fmt"
	_ "github.com/bmizerany/pq"
)

func init() {
	RegisterDialect("postgres", &Postgres{})
}

type Postgres struct{}

func (d *Postgres) Marker(pos int) string {
	return fmt.Sprintf("$%d", pos+1)
}

func (d *Postgres) SqlType(f interface{}, size int) string {
	switch f.(type) {
	case Id:
		return "bigserial"
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
	if size < 1 {
		size = 255
	}
	return fmt.Sprintf("varchar(%d)", size)
}

func (d *Postgres) Insert(hood *Hood, model *Model, query string, args ...interface{}) (Id, error, bool) {
	query = fmt.Sprintf("%v RETURNING %v", query, model.Pk.Name)
	var id int64
	err := hood.QueryRow(query, args...).Scan(&id)

	return Id(id), err, true
}

func (d *Postgres) StmtNotNull() string {
	return "NOT NULL"
}

func (d *Postgres) StmtDefault(s string) string {
	return fmt.Sprintf("DEFAULT %v", s)
}

func (d *Postgres) StmtPrimaryKey() string {
	return "PRIMARY KEY"
}

func (d *Postgres) StmtAutoIncrement() string {
	// postgres has not auto increment statement, uses SERIAL type
	return ""
}
