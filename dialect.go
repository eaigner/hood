package hood

type Dialect interface {
	Name() string                                          // dialect name
	Pk() string                                            // primary key
	Marker(pos int) string                                 // marker for a prepared statement, e.g. $1 or ?
	SqlType(f interface{}, size int, autoIncr bool) string // maps a go type to a db column type
	ScanOnInsert() bool
	StmtInsert(query string, model *Model) string
	StmtNotNull() string
	StmtDefault(s string) string
	StmtPrimaryKey() string
	StmtAutoIncrement() string
}
