package hood

type Dialect interface {
	Pk() string            // primary key
	Quote() rune           // quote rune
	Marker(pos int) string // marker for a prepared statement, e.g. $0 or ?
}
