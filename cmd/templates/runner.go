package main

import (
	"github.com/eaigner/hood"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type M struct{}

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
	for _, ts := range stamps {
		// TODO: check if was already run
		// TODO: pass real hood instance
		hood := &hood.Hood{}
		ups[ts].Call([]reflect.Value{v, reflect.ValueOf(hood)})
		downs[ts].Call([]reflect.Value{v, reflect.ValueOf(hood)})
	}
}
