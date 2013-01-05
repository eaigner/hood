package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	args := os.Args
	if len(args) >= 2 {
		c := strings.Split(args[1], ":")
		if len(c) == 2 {
			ns := c[0]
			cmd := c[1]
			rargs := args[2:]
			switch ns {
			case "create":
				create(cmd, rargs)
			case "db":
				db(cmd, rargs)
			}
		}
	}
	// fmt.Println("ARGS", args)
}

func create(cmd string, args []string) {
	switch cmd {
	case "migration":
		createMigration(args[0])
	}
}

func createMigration(name string) {
	if name == "" {
		return
	}
	dbDir := "db"
	err := os.MkdirAll(dbDir, 0777)
	if err != nil {
		log.Println(err.Error())
		return
	}
	ts := time.Now().Unix()
	mgrName := fmt.Sprintf("%s_%d", name, ts)
	fileName := fmt.Sprintf("db/%s.go", mgrName)
	template := fmt.Sprintf(migrationTemplate, mgrName, mgrName, mgrName)
	err = ioutil.WriteFile(fileName, []byte(template), 0644)
	if err != nil {
		log.Println(err.Error())
		return
	} else {
		log.Printf("created migration %s", fileName)
	}
}

func db(cmd string, args []string) {
	switch cmd {
	case "migrate":
		dbMigrate()
	}
}

func dbMigrate() {

}

var migrationTemplate = `package migrations

import (
	"github.com/eaigner/hood"
)

type %v struct {}

func (migration *%v) Up(hood *hood.Hood) {
	// implement
}

func (migration *%v) Down(hood *hood.Hood) {
	// implement
}`
