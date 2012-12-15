package hood

type Dialect interface {
	Pk() string                                                                           // primary key
	Quote(s string) string                                                                // value quotes
	Marker(pos int) string                                                                // marker for a prepared statement, e.g. $1 or ?
	SqlType(f interface{}, size int) string                                               // maps a go type to a db column type
	Insert(hood *Hood, model *Model, query string, args ...interface{}) (Id, error, bool) // if dialect requires custom insert, return true for last parameter (e.g. PostgreSQL requires RETURNING statement at the end or it won't return the inserted id)
	StmtNotNull() string
	StmtDefault(s string) string
	StmtPrimaryKey() string
	StmtAutoIncrement() string
}
