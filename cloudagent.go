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

type IntPercent struct {
	Percent int16 `json:"percent"`
}

type DiskInfo struct {
	System     IntPercent `json:"system"`
	Ephemeral  IntPercent `json:"ephemeral"`
	Persistent IntPercent `json:"persistent"`
}

type NetTraffic struct {
	Out float64 `json:"out"`
	In  float64 `json:"in"`
	Sum float64 `josn:"sum"`
}

type NTP struct {
	Offset    float64   `json:"offset"`
	TimeStamp time.Time `json:"timestamp"`
}

type SystemLoad struct {
	Load monit.Load   `json:"load"`
	Cpu  monit.Cpu    `json:"cpu"`
	Mem  monit.Memory `json:"mem"`
	Swap monit.Swap   `json:"swap"`
	Disk DiskInfo     `json:"disk"`
}

type JobInfo struct {
	Name      string
	Index     int
	Job_state string
}

type AgentInfo struct {
	Job       []JobInfo
	Vitals    SystemLoad
	Traffic   NetTraffic
	Ntp       NTP
	Address   string
	Cnt       int
	Eth0Stats sysfs.InterfaceStats
}

type HeartBeat struct {
	Job       string     `json:"job"`
	Index     int        `json:"index"`
	Job_state string     `json:"job_state"`
	Vitals    SystemLoad `json:"vitals"`
	Traffic   NetTraffic `json:"traffic"`
	Ntp       NTP        `json:"ntp"`
	Address   string     `json:"address"`
}

func GetMetrics(agentInfo *AgentInfo, conf *config.Config, interval int) {
	diskUsage := disk.GetDiskUsage(*conf)
	agentInfo.Vitals.Disk.System.Percent = int16(diskUsage["system"])
	agentInfo.Vitals.Disk.Ephemeral.Percent = int16(diskUsage["ephemeral"])
	agentInfo.Vitals.Disk.Persistent.Percent = int16(diskUsage["persistent"])

	monitStatus := monit.GetMonitStatus(conf)
	//num := 0
	for _, x := range monitStatus.Services {
		if strings.Contains(x.ServiceName, "system") {
			agentInfo.Vitals.Cpu = x.SysCpu
			agentInfo.Vitals.Load = x.SysLoad
			agentInfo.Vitals.Mem = x.SysMemory
			agentInfo.Vitals.Swap = x.SysSwap
			continue
		}
		tempJob := JobInfo{}
		tempJob.Name = x.ServiceName
		tempJob.Index = 1
		//TODO: use etcd lock to get real index.

		if x.Monitor == 0 {
			tempJob.Job_state = "starting"
		} else if x.Monitor == 1 {
			tempJob.Job_state = "running"
		} else {
			tempJob.Job_state = "not monitored"
		}
		agentInfo.Job = append(agentInfo.Job, tempJob)

	}
	sysFs, err := sysfs.NewRealSysFs()
	if err != nil {
		fmt.Printf("New Real SysFs error : %v \n", err)
	}

	if conf.Agent_id == "" {
		conf.Agent_id, err = sysFs.GetSystemUUID()
	}
	//	agentInfo.AgentID = conf.Agent_id

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
	agentInfo.Traffic.In = float64(newStats.RxBytes-preStats.RxBytes) / float64(interval)
	agentInfo.Traffic.Out = float64(newStats.TxBytes-preStats.TxBytes) / float64(interval)
	agentInfo.Traffic.Sum = agentInfo.Traffic.In + agentInfo.Traffic.Out
	agentInfo.Eth0Stats = newStats

	agentInfo.Ntp.TimeStamp = time.Now()
	agentInfo.Address = util.GetLocalIp()

}

func TransferMetrics(agentInfo *AgentInfo, conf *config.Config) {
	etcd.Connect(conf, util.GetLocalIp())
	nc, _ := nats.NatsConnect(conf.Mbus)
	for {
		GetMetrics(agentInfo, conf, 3)
		pubmessage := fmt.Sprintf("hm.agent.heartbeat.%s", conf.Agent_id)
		fmt.Println(pubmessage)
		for i := 0; i < len(agentInfo.Job); i++ {
			heartBeatInfo := HeartBeat{}
			heartBeatInfo.Address = agentInfo.Address
			heartBeatInfo.Index = agentInfo.Job[i].Index
			heartBeatInfo.Job = agentInfo.Job[i].Name
			heartBeatInfo.Job_state = agentInfo.Job[i].Job_state
			heartBeatInfo.Ntp = agentInfo.Ntp
			heartBeatInfo.Traffic = agentInfo.Traffic
			heartBeatInfo.Vitals = agentInfo.Vitals
			data, err := json.Marshal(heartBeatInfo)
			if err != nil {
				fmt.Printf("Json Marshal agentInfo error : %v\n", err)
			}
			nats.NatsPub(pubmessage, nc, data)
		}

		time.Sleep(3 * time.Second)
	}
}

func main() {
	config_path := flag.String("c", "", "config file path")
	daemon := flag.Bool("d", false, "run as cloud agent controller")
	flag.Parse()
	var (
		config_file string
		conf        *config.Config
		err         error
	)
	if !IsFile(*config_path) {
		conf = config.GetTemplate()
		fmt.Printf("No config file find in path: %s . Using template config file.\n", *config_path)
	} else {
		config_file = *config_path
		conf, err = config.GetConfig(config_file)
		if err != nil {
			fmt.Printf("get config error: %v\n", err)
			return
		}
	}
	if *daemon == false {
		agentInfo := &AgentInfo{}
		go TransferMetrics(agentInfo, conf)
	} else {

	}
	util.Trap(func() {})
}
