package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/wonderflow/cloudagent/config"
	"github.com/wonderflow/cloudagent/disk"
	"github.com/wonderflow/cloudagent/etcd"
	"github.com/wonderflow/cloudagent/monit"
	"github.com/wonderflow/cloudagent/nats"
	"github.com/wonderflow/cloudagent/sysfs"
	"github.com/wonderflow/cloudagent/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

/*
  Heartbeat payload example:
   {
     "job": "cloud_controller",
     "index": 3,
       "job_state":"running",
      "vitals": {
        "load": ["0.09","0.04","0.01"],
        "cpu": {"user":"0.0","sys":"0.0","wait":"0.4"},
        "mem": {"percent":"3.5","kb":"145996"},
        "swap": {"percent":"0.0","kb":"0"},
        "disk": {
          "system": {"percent" => "82"},
        "ephemeral": {"percent" => "5"},
        "persistent": {"percent" => "94"}
      },
       "traffic": {
           "in":{"max"=>123}
           "out":{"max"=>456}
           "sum":{"max"=434}
         }
       "ntp": {
           "offset": "-0.06423",
           "timestamp": "14 Oct 11:13:19"
       }
       "address": "127.0.0.1"
     }
*/

type DiskInfo struct {
	System     float64
	Ephemeral  float64
	Persistent float64
}

type NetTraffic struct {
	TxBytesPerSec      float64
	RxBytesPerSec      float64
	TxErrorBytesPerSec float64
	RxErrorBytesPerSec float64
}

type NTP struct {
	Offset    float64
	TimeStamp time.Time
}

type SystemLoad struct {
	Load monit.Load
	Cpu  monit.Cpu
	Mem  monit.Memory
	Swap monit.Swap
	Disk DiskInfo
}

type AgentInfo struct {
	AgentID     string
	Job         string
	Index       int
	Job_state   string
	System_load SystemLoad
	Traffic     NetTraffic
	Ntp         NTP
	Ip          string
	Eth0Stats   sysfs.InterfaceStats
	Cnt         int
}

func GetMetrics(agentInfo *AgentInfo, conf *config.Config, interval int) {
	agentInfo.AgentID = conf.Agent_id
	diskUsage := disk.GetDiskUsage(*conf)
	agentInfo.System_load.Disk.System = diskUsage["system"]
	agentInfo.System_load.Disk.Ephemeral = diskUsage["ephemeral"]
	agentInfo.System_load.Disk.Persistent = diskUsage["persistent"]

	monitStatus := monit.GetMonitStatus(conf)

	for _, x := range monitStatus.Services {
		if strings.Contains(x.ServiceName, "system") {
			agentInfo.System_load.Cpu = x.SysCpu
			agentInfo.System_load.Load = x.SysLoad
			agentInfo.System_load.Mem = x.SysMemory
			agentInfo.System_load.Swap = x.SysSwap
		}
	}
	sysFs, err := sysfs.NewRealSysFs()
	if err != nil {
		fmt.Printf("New Real SysFs error : %v \n", err)
	}
	//netDevices, err = sysfs.GetNetworkInfo(sysFs)
	var preStats sysfs.InterfaceStats
	if agentInfo.Cnt != 0 {
		var err error
		preStats, err = sysfs.GetNetworkStats("eth0", sysFs)
		if err != nil {
			fmt.Printf("GetNetworkStats error: '%v'\n", err)
		}
		time.Sleep(time.Second * time.Duration(interval))
	} else {
		preStats = agentInfo.Eth0Stats
	}
	newStats, err := sysfs.GetNetworkStats("eth0", sysFs)
	agentInfo.Cnt++
	agentInfo.Traffic.RxBytesPerSec = float64(newStats.RxBytes-preStats.RxBytes) / float64(interval)
	agentInfo.Traffic.TxBytesPerSec = float64(newStats.TxBytes-preStats.TxBytes) / float64(interval)
	agentInfo.Traffic.RxErrorBytesPerSec = float64(newStats.RxErrors-preStats.RxErrors) / float64(interval)
	agentInfo.Traffic.TxErrorBytesPerSec = float64(newStats.TxErrors-preStats.TxErrors) / float64(interval)
	agentInfo.Eth0Stats = newStats

	agentInfo.Ntp.TimeStamp = time.Now()
	agentInfo.Ip = util.GetLocalIp()

}

func TransferMetrics(agentInfo *AgentInfo, conf *config.Config) {
	nc, _ := nats.NatsConnect(conf.Mbus)
	for {
		GetMetrics(agentInfo, conf, 3)
		data, err := json.Marshal(agentInfo)
		if err != nil {
			fmt.Printf("Json Marshal agentInfo error : %v\n", err)
		}
		nats.NatsPub("cloudagent", nc, data)
		time.Sleep(3 * time.Second)
	}
}

func main() {
	config_path := flag.String("c", "", "config file path")
	flag.Parse()
	var (
		config_file string
		conf        *config.Config
		err         error
	)
	if !IsFile(*config_path) {
		conf = config.GetTemplate()
	} else {
		config_file = *config_path
		conf, err = config.GetConfig(config_file)
		if err != nil {
			fmt.Println("get config error")
			return
		}
	}
	agentInfo := &AgentInfo{}
	etcd.Connect(conf, util.GetLocalIp())

	go TransferMetrics(agentInfo, conf)
	util.Trap(func() {})
}
