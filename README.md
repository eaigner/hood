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
    
or you can pass an existing database and dialect to `hood.New(*sql.DB, hood.Dialect)`

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

	
## Example

	hd, err := hood.Open("postgres", "user=hood dbname=hood_test sslmode=disable")
	if err != nil {
		panic(err)
	}

	type Fruit struct {
		Id    hood.Id
		Name  string
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

	// Find
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