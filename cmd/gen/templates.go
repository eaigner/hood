package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	_, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	tplDir := "cmd/templates"
	fi, err := ioutil.ReadDir(tplDir)
	if err != nil {
		panic(err)
	}
	a := []string{
		"package main",
	}
	for _, f := range fi {
		name := strings.Split(f.Name(), ".")
		b, err := ioutil.ReadFile(tplDir + "/" + f.Name())
		if err != nil {
			panic(err)
		}
		s := fmt.Sprintf("var %sTmpl = `%s`", name[0], string(b))
		a = append(a, s)
	}
	err = ioutil.WriteFile("cmd/templates.go", []byte(strings.Join(a, "\r\n\r\n")), 0644)
	if err != nil {
		panic(err)
	}
}
