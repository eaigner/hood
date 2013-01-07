package main

import (
	"github.com/eaigner/hood"
	"fmt"
)

func (m *M) {{.Name}}_{{.Timestamp}}_Up(hood *hood.Hood) {
	// TODO: implement
	fmt.Println("up")
}

func (m *M) {{.Name}}_{{.Timestamp}}_Down(hood *hood.Hood) {
	// TODO: implement
	fmt.Println("down")
}