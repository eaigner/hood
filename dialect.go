package hood

type Dialect interface {
	Name() string                                          // dialect name
	Pk() string                                            // primary key
	Quote(s string) string                                 // quote string
	MarkerStartPos() int                                   // index for first marker
	Marker(pos int) string                                 // marker for a prepared statement, e.g. $1 or ?
	SqlType(f interface{}, size int, autoIncr bool) string // maps a go type to a db column type
}
