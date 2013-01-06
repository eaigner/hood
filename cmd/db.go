package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func db(cmd string, args []string) string {
	switch cmd {
	case "migrate":
		return dbMigrate()
	}
	return ""
}

func dbMigrate() string {
	// TODO: implement
	wd, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	mgrDir := wd + "/migrations"
	info, err := ioutil.ReadDir(mgrDir)
	if err != nil {
		return err.Error()
	}
	tmpDir, err := ioutil.TempDir("", "hood-migration-")
	if err != nil {
		return err.Error()
	}
	fmt.Println(tmpDir)
	// defer os.RemoveAll(tmpDir)
	for _, file := range info {
		_, err = copyFile(
			tmpDir+"/"+file.Name(),
			mgrDir+"/"+file.Name(),
		)
		if err != nil {
			return err.Error()
		}
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
