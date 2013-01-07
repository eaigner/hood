package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

func db(cmd string, args []string) string {
	switch cmd {
	case "migrate":
		return dbMigrate()
	}
	return ""
}

func dbMigrate() string {
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
