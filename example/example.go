package main

import (
	"hood"
	"time"
)

type Person struct {
	Id           hood.Id      // autoincrementing int field 'id'
	FirstName    string       `pk`                          // custom primary key field 'first_name'
	LastName     hood.VarChar `size(128)`                   // varchar field 'last_name' with size 128
	Tag          hood.VarChar `default('customer')`         // varchar field 'tag' with size 255
	CombinedTags hood.VarChar `size(128):default('orange')` // you can also combine tags
	Updated      time.Time    // timestamp field 'updated'
	Data         []byte       // data field 'data'
	IsAdmin      bool         // boolean field 'is_admin'
	Notes        string       // text field 'notes'
	// ... and other built in types (int, uint, float...)
}

func main() {

}
