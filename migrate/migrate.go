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
		if len(args) > 0 {
			dbDir := "db"
			err := os.MkdirAll(dbDir, 0777)
			if err != nil {
				log.Println(err.Error())
				return
			}
			ts := time.Now().Unix()
			name := fmt.Sprintf("db/%s_%d.go", args[0], ts)
			err = ioutil.WriteFile(name, []byte(migrationTemplate), 0644)
			if err != nil {
				log.Println(err.Error())
				return
			} else {
				log.Printf("created migration %s", name)
			}
		}
	}
}

func db(cmd string, args []string) {
	switch cmd {
	case "migrate":

	}
}

var migrationTemplate = `TODO: MIGRATION TEMPLATE`
