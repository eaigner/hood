package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	args := os.Args
	if len(args) >= 2 {
		c := strings.Split(args[1], ":")
		if len(c) == 2 {
			ns := c[0]
			cmd := c[1]
			rargs := args[2:]
			status := ""
			switch ns {
			case "create":
				status = create(cmd, rargs)
			case "db":
				status = db(cmd, rargs)
			}
			if status != "" {
				log.Printf("\x1b[32m%s\x1b[0m", status)
			}
		}
	} else {
		log.Println("invalid arguments")
	}
}
