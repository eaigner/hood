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

type M struct{}

type environments map[string]config
type config map[string]string

func main() {
	v := reflect.ValueOf(&M{})
	numMethods := v.NumMethod()
	stamps := make([]int, 0, numMethods)
	ups := make(map[int]reflect.Value)
	downs := make(map[int]reflect.Value)
	for i := 0; i < numMethods; i++ {
		method := v.Type().Method(i)
		chunks := strings.Split(method.Name, "_")
		if l := len(chunks); l >= 3 {
			ts, _ := strconv.Atoi(chunks[l-2])
			direction := chunks[l-1]
			if strings.ToLower(direction) == "up" {
				ups[ts] = method.Func
				stamps = append(stamps, ts)
			} else {
				downs[ts] = method.Func
			}
		}
	}
	sort.Ints(stamps)
	env := readConfig()
	fmt.Println(env)
	for _, ts := range stamps {
		// TODO: check if was already run
		// TODO: pass real hood instance
		hood := &hood.Hood{}
		ups[ts].Call([]reflect.Value{v, reflect.ValueOf(hood)})
		downs[ts].Call([]reflect.Value{v, reflect.ValueOf(hood)})
	}
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
