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
}

func create(cmd string, args []string) {
	switch cmd {
	case "migration":
		if len(args) > 0 {
			createMigration(args[0])
		}
	}
}

func createMigration(name string) {
	if name == "" {
		return
	}
	do := func() string {
		dbDir := "migrations"

		err := os.MkdirAll(dbDir, 0777)
		if err != nil {
			return err.Error()
		}
		ts := time.Now().Unix()
		fileName := fmt.Sprintf("%s/%d_%s.go", dbDir, ts, name)
		structName := fmt.Sprintf("%s_%d", name, ts)
		template := fmt.Sprintf(migrationTemplate, structName, structName, structName)
		err = ioutil.WriteFile(fileName, []byte(template), 0644)
		if err != nil {
			return err.Error()
		}
		return fmt.Sprintf("created migration %s", fileName)
	}
	log.Println(do())
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
