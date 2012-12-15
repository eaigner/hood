## Hood

Hood is a database agnostic ORM for Go. It was written with following points in mind:

- Chainable API
- Transaction support
- Database dialect interface
- Clean and testable codebase

Dialects currently supported

- Postgres

## 1. Open

If the dialect is registered, you can open the database directly using

    hd, err := hood.Open("postgres", "user=<username> dbname=<database>")
    
or you can pass an existing `*sql.DB` object and dialect to `hood.New(*sql.DB, hood.Dialect)`

	hd, err := hood.New(db, &Postgres{})
	
## 2. Save

## 3. Find

## 4. Delete