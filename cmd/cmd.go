package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"
)

var (
	kDbDir      = "db"
	kMgrDir     = "migrations"
	kConfFile   = "config.json"
	kRunnerFile = "runner.go"
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
	err = os.MkdirAll(path.Join(wd, kDbDir), 0777)
	if err != nil {
		panic(err)
	}
	confPath := path.Join(kDbDir, kConfFile)
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
	err = os.MkdirAll(path.Join(wd, kDbDir, kMgrDir), 0777)
	if err != nil {
		panic(err)
	}
	ts := time.Now().Unix()
	fileName := path.Join(kDbDir, kMgrDir, fmt.Sprintf("%d_%s.go", ts, name))
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

func db(cmd string, args []string) string {
	switch cmd {
	case "migrate":
		return dbMigrate(true)
	case "rollback":
		return dbMigrate(false)
	}
	return ""
}

func dbMigrate(up bool) string {
	wd, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	mgrDir := path.Join(wd, kDbDir, kMgrDir)
	info, err := ioutil.ReadDir(mgrDir)
	if err != nil {
		return err.Error()
	}
	tmpDir, err := ioutil.TempDir("", "hood-migration-")
	if err != nil {
		return err.Error()
	}
	defer os.RemoveAll(tmpDir)
	files := []string{}
	for _, file := range info {
		dstFile := path.Join(tmpDir, file.Name())
		_, err = copyFile(
			dstFile,
			path.Join(mgrDir, file.Name()),
		)
		if err != nil {
			return err.Error()
		}
		files = append(files, dstFile)
	}
	main := path.Join(tmpDir, kRunnerFile)
	err = ioutil.WriteFile(main, []byte(runnerTmpl), 0666)
	if err != nil {
		return err.Error()
	}
	files = append(files, main)
	cmd := exec.Command("go", "run")
	cmd.Args = append(cmd.Args, files...)
	cmd.Args = append(cmd.Args, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return err.Error()
	}
	return ""
}

func copyFile(dst, src string) (int64, error) {
	sf, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer df.Close()
	return io.Copy(df, sf)
}
