package hood

import "reflect"

type Dialect interface {
	// Marker returns the dialect specific markers for prepared statements,
	// for instance $1, $2, ... and so on. The position starts at 0.
	Marker(pos int) string

	// SqlType returns the SQL type for the provided interface type. The size
	// parameter delcares the data size for the column (e.g. for VARCHARs).
	SqlType(f interface{}, size int) string

	// ValueToField converts from an SQL Value to the coresponding interface Value.
	// It is the opposite of SqlType, in a sense.
	// For example: time.Time objects needs to be marshalled back and forth
	// as Strings for databases that don't have a native "time" type.
	ValueToField(value reflect.Value, field reflect.Value) error

	// Insert takes the generated query and modifies it. E.g. Postgres does not
	// return the inserted IDs after executing INSERT, unless a RETURNING
	// keyword is appended. If a dialect needs a custom INSERT, it should return
	// implemented == true.
	Insert(hood *Hood, model *Model, query string, args ...interface{}) (id Id, err error, implemented bool)

	// KeywordNotNull returns the dialect specific keyword for 'NOT NULL'.
	KeywordNotNull() string

	// KeywordDefault returns the dialect specific keyword for 'DEFAULT'.
	KeywordDefault(s string) string

	// KeywordPrimaryKey returns the dialect specific keyword for 'PRIMARY KEY'.
	KeywordPrimaryKey() string

	// KeywordAutoIncrement returns the dialect specific keyword for 'AUTO_INCREMENT'.
	KeywordAutoIncrement() string
}
