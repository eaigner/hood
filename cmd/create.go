package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"
	"time"
)

var (
	dbDir  = "db"
	mgrDir = "migrations"
)

func create(cmd string, args []string) string {
	switch cmd {
	case "migration":
		if len(args) > 0 {
			return createMigration(args[0])
		}
	case "config":
		return createDbConfig()
	}
	return ""
}

func createDbConfig() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(path.Join(wd, dbDir), 0777)
	if err != nil {
		panic(err)
	}
	confPath := path.Join(dbDir, "config.json")
	err = ioutil.WriteFile(path.Join(wd, confPath), []byte(confTmpl), 0666)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("created db configuration '%s'", confPath)
}

func createMigration(name string) string {
	if name == "" {
		return "invalid migration name"
	}
	wd, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	err = os.MkdirAll(path.Join(wd, dbDir, mgrDir), 0777)
	if err != nil {
		panic(err)
	}
	ts := time.Now().Unix()
	fileName := path.Join(dbDir, mgrDir, fmt.Sprintf("%d_%s.go", ts, name))
	f, err := os.OpenFile(path.Join(wd, fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	err = migrationTemplate.Execute(f, &Migration{
		Timestamp: ts,
		Name:      name,
	})
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("created migration '%s'", fileName)
}

type Migration struct {
	Timestamp int64
	Name      string
}

var migrationTemplate = template.Must(template.New("migration").Parse(migrationTmpl))

var confTmpl = `{
  "development": {
    "driver": "",
    "source": ""
  },
  "production": {
    "driver": "",
    "source": ""
  },
  "test": {
    "driver": "",
    "source": ""
  }
}`
