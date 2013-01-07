package main

import (
	"encoding/json"
	"github.com/eaigner/hood"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

import (
	"fmt"
)

type (
	M          struct{}
	Migrations struct {
		Id      hood.Id
		Current int
	}
	environments map[string]config
	config       map[string]string
)

func main() {
	// Get up/down migration methods
	v := reflect.ValueOf(&M{})
	numMethods := v.NumMethod()
	stamps := make([]int, 0, numMethods)
	ups := make(map[int]reflect.Method)
	downs := make(map[int]reflect.Method)
	for i := 0; i < numMethods; i++ {
		method := v.Type().Method(i)
		chunks := strings.Split(method.Name, "_")
		if l := len(chunks); l >= 3 {
			ts, _ := strconv.Atoi(chunks[l-2])
			direction := chunks[l-1]
			if strings.ToLower(direction) == "up" {
				ups[ts] = method
				stamps = append(stamps, ts)
			} else {
				downs[ts] = method
			}
		}
	}
	sort.Ints(stamps)
	// Get config for set environment
	env := "development"
	if x := os.Getenv("HOOD_ENV"); x != "" {
		env = x
	}
	cfg := readConfig()[env]
	// Open hood
	hd, err := hood.Open(cfg["driver"], cfg["source"])
	if err != nil {
		panic(err)
	}
	// Check migration table
	err = hd.CreateTableIfNotExists(&Migrations{})
	if err != nil {
		panic(err)
	}
	var rows []Migrations
	err = hd.Find(&rows)
	if err != nil {
		panic(err)
	}
	info := Migrations{}
	if len(rows) > 0 {
		info = rows[0]
	}
	runCount := 0
	for _, ts := range stamps {
		if ts > info.Current {
			tx := hd.Begin()
			method := ups[ts]
			method.Func.Call([]reflect.Value{v, reflect.ValueOf(tx)})
			info.Current = ts
			tx.Save(&info)
			err = tx.Commit()
			if err != nil {
				panic(err)
			} else {
				runCount++
				fmt.Printf("applied %s\n", method.Name)
			}
		}
		// downs[ts].Call([]reflect.Value{v, reflect.ValueOf(hd)})
	}
	fmt.Printf("applied %d migrations\n", runCount)
}

func readConfig() environments {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	sf, err := os.Open(path.Join(wd, "db", "config.json"))
	if err != nil {
		panic(err)
	}
	defer sf.Close()
	dec := json.NewDecoder(sf)
	var env environments
	err = dec.Decode(&env)
	if err != nil {
		panic(err)
	}
	return env
}
