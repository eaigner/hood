## Hood

Hood is a database agnostic ORM for Go. It was written with following points in mind:

- Chainable API
- Transaction support
- Database dialect interface
- Clean and testable codebase

Dialects currently supported

- Postgres

## Open

If the dialect is registered, you can open the database directly using

    hd, err := hood.Open("postgres", "user=<username> dbname=<database>")
    
or you can pass an existing `*sql.DB` object and dialect to `hood.New(*sql.DB, hood.Dialect)`

	hd, err := hood.New(db, &Postgres{})
	
## Models

Tables can be declared using the following syntax (please not that this table is only for demonstration purposes and would not produce valid SQL since it has 2 primary keys)

	type Person struct {
		Id           Id        // autoincrementing int field 'id'
		FirstName    string    `pk`                          // custom primary key field 'first_name'
		LastName     VarChar   `size(128)`                   // varchar field 'last_name' with size 128
		Tag          VarChar   `default('customer')`         // varchar field 'tag' with size 255
		CombinedTags VarChar   `size(128):default('orange')` // you can also combine tags
		Updated      time.Time // timestamp field 'updated'
		Data         []byte    // data field 'data'
		IsAdmin      bool      // boolean field 'is_admin'
		Notes        string    // text field 'notes'
		// ... and other built in types (int, uint, float...)
	}

	
## Save

## Find

## Delete