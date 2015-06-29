package disk

import (
	//"fmt"
	"github.com/cloudfoundry/gosigar"
	"github.com/wonderflow/cloudagent/config"
)

func GetUsageFromDir(dir_name string) float64 {
	usage := sigar.FileSystemUsage{}
	usage.Get(dir_name)
	return usage.UsePercent()
}

func GetDiskUsage(conf config.Config) map[string]float64 {
	usage := map[string]float64{}
	usage["system"] = GetUsageFromDir("/")
	usage["basedir"] = GetUsageFromDir(conf.Base_dir)
	usage["ephemeral"] = GetUsageFromDir(conf.Base_dir + "/data")
	usage["persistent"] = GetUsageFromDir(conf.Base_dir + "/store")
	return usage
}
