## Hood

Hood is a database agnostic ORM for Go developed by [@eaignr](https://twitter.com/eaignr). It was written with following points in mind:

- Chainable API
- Transaction support
- Model validations
- Database dialect interface
- No implicit fields
- Clean and testable codebase

Dialects currently supported

- **Postgres** using [github.com/bmizerany/pq](https://github.com/bmizerany/pq)
- **Sqlite3** ** (by [lbolla](https://github.com/lbolla)) using [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)

** not registered by default, requires some packages installed on the system

Adding a dialect is as simple as copying the original `postgres.go`, replacing the statement and field values with the new dialect versions.


## Documentation

[GoDoc](http://godoc.org/github.com/eaigner/hood)

## Open

If the dialect is registered, you can open the database directly using

    hd, err := hood.Open("postgres", "user=<username> dbname=<database>")
    
or you can pass an existing database and dialect to `hood.New(*sql.DB, hood.Dialect)`

	hd, err := hood.New(db, &Postgres{})
	
## Schemas

Schemas can be declared using the following syntax (only for demonstration purposes, would not produce valid SQL since it has 2 primary keys)

```go
type Person struct {
	// Auto-incrementing int field 'id'
	Id hood.Id

	// Custom primary key field 'first_name', with presence validation
	FirstName string `sql:"pk" validate:"presence"`

	// Varchar field 'last_name' with size 128, NOT NULL
	LastName hood.VarChar `sql:"size(128),notnull"`

	// Varchar field 'tag' with size 255, default value 'customer'
	Tag hood.VarChar `sql:"default('customer')"`

	// You can also combine tags, default value 'orange'
	CombinedTags hood.VarChar `sql:"size(128),default('orange')"`
	Updated      time.Time    // timestamp field 'updated'
	Data         []byte       // data field 'data'
	IsAdmin      bool         // boolean field 'is_admin'
	Notes        string       // text field 'notes'

	// Validates number range
	Balance int `validate:"range(10:20)"`

	// ... and other built in types (int, uint, float...)
}
```

Schema creation is completely optional, you can use any other tool you like.	
## Validation

Besides the `sql:` struct tag, you can specify a `validate:` tag for model validation:

- `presence` validates that a field is set
- `len(min:max)` validates that a `string` or `VarChar` field’s length lies within the specified range
	- `len(min:)` validates that it has the specified min length, 
	- `len(:max)` or max length
- `range(min:max)` validates that an `int` value lies in the specific range
	- `range(min:)` validates that it has the specified min value,
	- `range(:max)` or max value

You can also define multiple validations on one field, e.g. `validate:"len(:12),presence"`

For more complex validations you can use custom validation methods. The methods
are added to the schema and must start with `Validate` and return an `error`.

For example:

```go
func (u *User) ValidateUsername() error {
	rx := regexp.MustCompile(`[a-z0-9]+`)
	if !rx.MatchString(u.Name) {
		return NewValidationError(1, "username contains invalid characters")
	}
	return nil
}
```

## Hooks

You can add hooks to a model to run on a specific action like so:

```go
func (u *User) BeforeUpdate() error {
	u.Updated = time.Now()
	return nil
}
```

If the hooks returns an error on a `Before-` action it **is not performed**!

The following hooks are defined:

- `Before/AfterSave`
- `Before/AfterInsert`
- `Before/AfterUpdate`
- `Before/AfterDelete`

## Example

```go

package main

import (
	"hood"
)

func main() {
	// Open a DB connection, use New() alternatively for unregistered dialects
	hd, err := hood.Open("postgres", "user=hood dbname=hood_test sslmode=disable")
	if err != nil {
		panic(err)
	}

	// Create a table
	type Fruit struct {
		Id    hood.Id
		Name  string `validate:"presence"`
		Color string
	}

	err = hd.CreateTable(&Fruit{})
	if err != nil {
		panic(err)
	}

	fruits := []Fruit{
		Fruit{Name: "banana", Color: "yellow"},
		Fruit{Name: "apple", Color: "red"},
		Fruit{Name: "grapefruit", Color: "yellow"},
		Fruit{Name: "grape", Color: "green"},
		Fruit{Name: "pear", Color: "yellow"},
	}

	// Start a transaction
	tx := hd.Begin()

	ids, err := tx.SaveAll(&fruits)
	if err != nil {
		panic(err)
	}

	fmt.Println("inserted ids:", ids) // [1 2 3 4 5]

	// Commit changes
	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	// Ids are automatically updated
	if fruits[0].Id != 1 || fruits[1].Id != 2 || fruits[2].Id != 3 {
		panic("id not set")
	}

	// If an id is already set, a call to save will result in an update
	fruits[0].Color = "green"

	ids, err = hd.SaveAll(&fruits)
	if err != nil {
		panic(err)
	}

	fmt.Println("updated ids:", ids) // [1 2 3 4 5]

	if fruits[0].Id != 1 || fruits[1].Id != 2 || fruits[2].Id != 3 {
		panic("id not set")
	}

	// Let's try to save a row that does not satisfy the required validations
	_, err = hd.Save(&Fruit{})
	if err == nil || err.Error() != "value not set" {
		panic("does not satisfy validations, should not save")
	}

	// Find
	//
	// The markers are db agnostic, so you can always use '?'
	// e.g. in Postgres they are replaced with $1, $2, ...
	var results []Fruit
	err = hd.Where("color = ?", "green").OrderBy("name").Limit(1).Find(&results)
	if err != nil {
		panic(err)
	}

	fmt.Println("results:", results) // [{1 banana green}]

	// Delete
	ids, err = hd.DeleteAll(&results)
	if err != nil {
		panic(err)
	}

	fmt.Println("deleted ids:", ids) // [1]

	results = nil
	err = hd.Find(&results)
	if err != nil {
		panic(err)
	}

	fmt.Println("results:", results) // [{2 apple red} {3 grapefruit yellow} {4 grape green} {5 pear yellow}]

	// Drop
	hd.DropTable(&Fruit{})
}
```
