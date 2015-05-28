package main

import (
	//"fmt"
	"github.com/cloudfoundry/gosigar"
)

func GetUsageFromDir(dir_name string) float64 {
	usage := sigar.FileSystemUsage{}
	usage.Get(dir_name)
	return usage.UsePercent()
}

func GetDiskUsage(conf Config) map[string]float64 {
	usage := map[string]float64{}
	usage["system"] = GetUsageFromDir("/")
	usage["basedir"] = GetUsageFromDir(conf.Base_dir)
	usage["data"] = GetUsageFromDir(conf.Base_dir + "/data")
	usage["store"] = GetUsageFromDir(conf.Base_dir + "/store")
	return usage
}
