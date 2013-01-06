package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

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
		pwd, err := os.Getwd()
		if err != nil {
			return err.Error()
		}
		migrationsDir := pwd + "/migrations"
		err = os.MkdirAll(migrationsDir, 0777)
		if err != nil {
			return err.Error()
		}
		ts := time.Now().Unix()
		fileName := fmt.Sprintf("%s/%d_%s.go", migrationsDir, ts, name)
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
