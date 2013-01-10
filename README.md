## Index

- [Overview](#overview)
- [Documentation](#documentation)
- [Opening a Database](#opening-a-database)
- [Schemas](#schemas)
- [Migrations](#migrations)
- [Validation](#validation)
- [Hooks](#hooks)
- [Basic Example](#basic-example)
- [Contributors](hood/graphs/contributors)

## Overview

Hood is a database agnostic ORM for Go developed by [@eaignr](https://twitter.com/eaignr). It was written with following points in mind:

- Chainable API
- Transaction support
- Migration and schema generation
- Model validations
- Model event hooks
- Database dialect interface
- No implicit fields
- Clean and testable codebase

Dialects currently supported

- **Postgres** using [github.com/bmizerany/pq](https://github.com/bmizerany/pq)
- **Sqlite3** ** (partial, by [lbolla](https://github.com/lbolla)) using [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)

** not registered by default, requires some packages installed on the system

Adding a dialect is simple. Just create a new file named `<dialect_name>.go` and the corresponding struct type, and mixin the `Base` dialect. Then implement the methods that are specific to the new dialect (for an example see `postgres.go`).

## Documentation

You can find the documentation over at [GoDoc](http://godoc.org/github.com/eaigner/hood)

## Opening a Database

If the dialect is registered, you can open the database directly using

    hd, err := hood.Open("postgres", "user=<username> dbname=<database>")
    
or you can pass an existing database and dialect to `hood.New(*sql.DB, hood.Dialect)`

    hd := hood.New(db, NewPostgres())
	
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

	// Create named multi column indexes
	TagIndex  hood.Index       `sql:"columns(tag)"`
	NameIndex hood.UniqueIndex `sql:"columns(first_name:last_name)"`

	// ... and other built in types (int, uint, float...)
}
```

Schema creation is completely optional, you can use any other tool you like.

The following built in field properties are defined (via `sql:` tag):

- `pk` the field is a primary key
- `notnull` the field must be NOT NULL
- `size(x)` the field must have the specified size, e.g. for varchar `size(128)`
- `default(x)` the field has the specified default value, e.g. `default(5)` or `default('orange')`
- `-` ignores the field

## Migrations

To use migrations, you first have to install the `hood` tool. To do that run the following:

    go get github.com/eaigner/hood
    cd $GOPATH/src/github.com/eaigner/hood
    ./install.sh

Assuming you have your `$GOPATH/bin` directory in your `PATH`, you can now invoke the hood tool with `hood`.
Before we can use migrations we have to create a database configuration file first. To do this type

    hood create:config

This command will create a `db/config.json` file relative to your current directory. It will look something like this:

```javascript
{
  "development": {
    "driver": "",
    "source": ""
  },
  "production": {
    "driver": "",
    "source": ""
  },
  "test": {
    "driver": "",
    "source": ""
  }
}
```

Populate it with your database credentials. The `driver` and `source` fields are the strings you would pass
to the `sql.Open(2)` function. Now hood knows about our database, so let's create our first migration with

    hood create:migration CreateUserTable

and another one

		hood create:migration AddUserNameIndex

This command creates new migrations in `db/migrations`. Next we have to populate the
generated migrations `Up` (and `Down`) methods like so:

```go
func (m *M) CreateUserTable_1357605106_Up(hd *hood.Hood) {
	type Users struct {
		Id		hood.Id
		First	string
		Last	string
	}
	hd.CreateTable(&Users{})
}
```

```go
func (m *M) AddUserNameIndex_1357605116_Up(hd *hood.Hood) {
	hd.CreateIndex("users", struct{
		NameIndex hood.UniqueIndex `sql:"columns(first:last)"`
	}{})
}
```

The passed in `hood` instance is a transaction that will be committed after this method.

Now we can run migrations with

	hood db:migrate

and roll back with

	hood db:rollback

If you want to run a environment configuration other than `development`, you have to set an environment variable first like this:

	export HOOD_ENV=production

A `schema.go` file is **automatically generated** in the `db` directory. This file reflects the
current state of the database! It will look like this:

```go
type Users struct {
	Id		hood.Id
	First	string
	Last	string

	// Indexes
	NameIndex	hood.UniqueIndex	`sql:"columns(first:last)"`
}
```

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

If the hook returns an error on a `Before-` action it **is not performed**!

The following hooks are defined:

- `Before/AfterSave`
- `Before/AfterInsert`
- `Before/AfterUpdate`
- `Before/AfterDelete`

## Basic Example

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
