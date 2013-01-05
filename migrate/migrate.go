package main

import (
	"fmt"
	// "github.com/eaigner/hood"
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
	fileName := fmt.Sprintf("db/%s_%d.go", name, ts)
	err = ioutil.WriteFile(fileName, []byte(migrationTemplate), 0644)
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

var migrationTemplate = `TODO: MIGRATION TEMPLATE`
