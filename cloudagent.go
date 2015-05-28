package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IsFile(file string) bool {
	f, e := os.Stat(file)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

func GetCurrPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	splitstring := strings.Split(path, "/")
	size := len(splitstring)
	splitstring = splitstring[0 : size-1]
	ret := strings.Join(splitstring, "/")
	return ret
}

func main() {
	config_path := flag.String("c", "", "config file path")
	flag.Parse()
	var config_file string
	if !IsFile(*config_path) {
		//fmt.Println("config file doesn't exist.")
		curpath := GetCurrPath()
		config_file = curpath + "/config.yml"
	} else {
		config_file = *config_path
	}
	conf, err := GetConfig(config_file)
	if err != nil {
		fmt.Println("get config error")
		return
	} else {
		fmt.Println(conf.Agent_id)
	}
	//fmt.Println(GetDiskUsage(*myconfig))
	GetMonitStatus(conf)
}
