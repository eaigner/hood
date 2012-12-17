package main

import (
	"fmt"
	"hood"
	"time"
)

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
