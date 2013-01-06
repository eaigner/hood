package main

import (
	"fmt"
	"os"
	"text/template"
	"time"
)

func create(cmd string, args []string) string {
	switch cmd {
	case "migration":
		if len(args) > 0 {
			return createMigration(args[0])
		}
	}
	return ""
}

func createMigration(name string) string {
	if name == "" {
		return "invalid migration name"
	}
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
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err.Error()
	}
	defer f.Close()
	err = tmpl.Execute(f, &Migration{
		Timestamp: ts,
		Name:      name,
	})
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("created migration %s", fileName)
}

type Migration struct {
	Timestamp int64
	Name      string
}

var tmpl = template.Must(template.New("migration").Parse(`package main

import (
	"github.com/eaigner/hood"
)

func (m *Migration) {{.Name}}_{{.Timestamp}}_Up(hood *hood.Hood) {
	// TODO: implement
}

func (m *Migration) {{.Name}}_{{.Timestamp}}_Down(hood *hood.Hood) {
	// TODO: implement
}`))
