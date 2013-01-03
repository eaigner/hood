package hood

import "reflect"

type Dialect interface {
	// Marker returns the dialect specific markers for prepared statements,
	// for instance $1, $2, ... and so on. The position starts at 0.
	Marker(pos int) string

	// Next marker returns the dialect specific marker for pos and increments
	// the position by one.
	NextMarker(pos *int) string

	// SqlType returns the SQL type for the provided interface type. The size
	// parameter delcares the data size for the column (e.g. for VARCHARs).
	SqlType(f interface{}, size int) string

	// ValueToField converts from an SQL Value to the coresponding interface Value.
	// It is the opposite of SqlType, in a sense.
	// For example: time.Time objects needs to be marshalled back and forth
	// as Strings for databases that don't have a native "time" type.
	ValueToField(value reflect.Value, field reflect.Value) error

	// QuerySql returns the resulting query sql and attributes.
	QuerySql(hood *Hood) (sql string, args []interface{})

	// Insert inserts the values in model and returns the inserted rows Id.
	Insert(hood *Hood, model *Model) (Id, error)

	// InsertSql returns the sql for inserting the passed model.
	InsertSql(hood *Hood, model *Model) (sql string, args []interface{})

	// Update updates the values in the specified model and returns the
	// updated rows Id.
	Update(hood *Hood, model *Model) (Id, error)

	// UpdateSql returns the sql for updating the specified model.
	UpdateSql(hood *Hood, model *Model) (string, []interface{})

	// Delete drops the row matching the primary key of model and returns the affected Id.
	Delete(hood *Hood, model *Model) (Id, error)

	// DeleteSql returns the sql for deleting the row matching model's primary key.
	DeleteSql(hood *Hood, model *Model) (string, []interface{})

	// CreateTable creates the table specified in model.
	CreateTable(hood *Hood, model *Model) error

	// CreateTableSql returns the sql for creating a table.
	CreateTableSql(hood *Hood, model *Model) string

	// DropTable drops the specified table.
	DropTable(hood *Hood, table string) error

	// DropTableSql returns the sql for dropping the specified table.
	DropTableSql(hood *Hood, table string) string

	// RenameTable renames the specified table.
	RenameTable(hood *Hood, from, to string) error

	// RenameTableSql returns the sql for renaming the specified table.
	RenameTableSql(hood *Hood, from, to string) string

	// AddColumn adds the columns to the corresponding table.
	AddColumn(hood *Hood, table string, column *Field) error

	// AddColumnSql returns the sql for adding the specified column in table.
	AddColumnSql(hood *Hood, table string, column *Field) string

	// RenameColumn renames a table column in the specified table.
	RenameColumn(hood *Hood, table, from, to string) error

	// RenameColumnSql returns the sql for renaming the specified column in table.
	RenameColumnSql(hood *Hood, table, from, to string) string

	// KeywordNotNull returns the dialect specific keyword for 'NOT NULL'.
	KeywordNotNull() string

	// KeywordDefault returns the dialect specific keyword for 'DEFAULT'.
	KeywordDefault(s string) string

	// KeywordPrimaryKey returns the dialect specific keyword for 'PRIMARY KEY'.
	KeywordPrimaryKey() string

	// KeywordAutoIncrement returns the dialect specific keyword for 'AUTO_INCREMENT'.
	KeywordAutoIncrement() string
}
