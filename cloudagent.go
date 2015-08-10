package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	rawetcd "github.com/coreos/go-etcd/etcd"
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
	"strconv"
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
  Heartbeat system payload example:
   {
     "job": "system",
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

/*
  Heartbeat process payload example:
   {
     "job": "cloud_controller",
     "index": 3,
     "job_state":"running",
     "vitals": {
       "cpu" :{"percenttotal" =>"0.4"},
       "mem": { "percent" => "0.5", "kb" => "21212" },
       "process": { "status" => "0", "monitor" => "1","uptime"=>"2579142","children"=>"0"}
      },
     "ntp": {
           "offset": "-0.06423",
           "timestamp": "14 Oct 11:13:19"
       }
     "address": "127.0.0.1"
     }
*/

type FloatPercent struct {
	Percent float64 `json:"percent"`
}

type PercentTotal struct {
	Percent float64 `json:"percenttotal"`
}

type DiskInfo struct {
	System     FloatPercent `json:"system"`
	Ephemeral  FloatPercent `json:"ephemeral"`
	Persistent FloatPercent `json:"persistent"`
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
	Load []float64    `json:"load"`
	Cpu  monit.Cpu    `json:"cpu"`
	Mem  monit.Memory `json:"mem"`
	Swap monit.Swap   `json:"swap"`
	Disk DiskInfo     `json:"disk"`
}

type ProcessInfo struct {
	Status   int `json:"status"`
	Monitor  int `json:"monitor"`
	Uptime   int `json:"uptime"`
	Children int `json:"children"`
}

type ProcessVital struct {
	Cpu     PercentTotal `json:"cpu"`
	Mem     monit.Memory `json:"mem"`
	Process ProcessInfo  `json:"process"`
}

type JobInfo struct {
	Name        string
	Index       int
	Job_state   string
	ProcessData ProcessVital
	Type        int
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

type SysHeartBeat struct {
	Job       string     `json:"job"`
	Index     int        `json:"index"`
	Job_state string     `json:"job_state"`
	Vitals    SystemLoad `json:"vitals"`
	Traffic   NetTraffic `json:"traffic"`
	Ntp       NTP        `json:"ntp"`
	Address   string     `json:"address"`
	Cores     int        `json:"cores"`
}
type ProcessHeartBeat struct {
	Job       string       `json:"job"`
	Index     int          `json:"index"`
	Job_state string       `json:"job_state"`
	Vitals    ProcessVital `json:"vitals"`
	Ntp       NTP          `json:"ntp"`
	Address   string       `json:"address"`
}

//monit type
const (
	FILE_SYSTEM = 0
	DIRECTORY   = 1
	FILE        = 2
	PROCESS     = 3
	REMOTE_HOST = 4
	SYSTEM      = 5
	FIFO        = 6
)

var localIP string

func init() {
	localIP = util.GetLocalIp()
}

func GetMetrics(agentInfo *AgentInfo, conf *config.Config, interval int, etcdclient *rawetcd.Client) {
	diskUsage := disk.GetDiskUsage(*conf)
	agentInfo.Vitals.Disk.System.Percent = diskUsage["system"]
	agentInfo.Vitals.Disk.Ephemeral.Percent = diskUsage["ephemeral"]
	agentInfo.Vitals.Disk.Persistent.Percent = diskUsage["persistent"]

	monitStatus := monit.GetMonitStatus(conf)

	for _, x := range monitStatus.Services {
		tempJob := JobInfo{}

		tempJob.Type = x.Type
		if x.Monitor == 0 {
			tempJob.Job_state = "starting"
		} else if x.Monitor == 1 {
			tempJob.Job_state = "running"
		} else {
			tempJob.Job_state = "not monitored"
		}

		if x.Type == SYSTEM {
			agentInfo.Vitals.Cpu = x.SysCpu
			agentInfo.Vitals.Load = append(agentInfo.Vitals.Load, x.SysLoad.Avg01)
			agentInfo.Vitals.Load = append(agentInfo.Vitals.Load, x.SysLoad.Avg05)
			agentInfo.Vitals.Load = append(agentInfo.Vitals.Load, x.SysLoad.Avg15)
			agentInfo.Vitals.Mem = x.SysMemory
			agentInfo.Vitals.Swap = x.SysSwap
			tempJob.Name = localIP
			tempJob.Index = 0
		} else {
			tempJob.ProcessData.Cpu.Percent = x.CpuPercent
			tempJob.ProcessData.Mem.Kb = x.Memory.Kb
			tempJob.ProcessData.Mem.Percent = x.Memory.Percent
			tempJob.ProcessData.Process.Children = x.Children
			tempJob.ProcessData.Process.Monitor = x.Monitor
			tempJob.ProcessData.Process.Status = x.Status
			tempJob.ProcessData.Process.Uptime = x.Uptime
			tempJob.Name = x.ServiceName

			index, err := etcd.GetIndex(etcdclient, conf.Etcd_dir, tempJob.Name, localIP)
			if err != nil {
				fmt.Printf("etcd get index error: %v\n", err)
				index = 0
			}
			tempJob.Index = index
		}

		agentInfo.Job = append(agentInfo.Job, tempJob)

	}
	sysFs, err := sysfs.NewRealSysFs()
	if err != nil {
		fmt.Printf("New Real SysFs error : %v \n", err)
	}

	if conf.Agent_id == "" || conf.Agent_id == "test" {
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
	agentInfo.Address = localIP

}

func TransferMetrics(agentInfo *AgentInfo, conf *config.Config) {
	etcdclient := etcd.Connect(conf, localIP)
	etcd.EtcdHup(etcdclient, conf, localIP)
	nc, _ := nats.NatsConnect(conf.Mbus)
	cmd := exec.Command("/bin/sh", "-c", "cat /proc/cpuinfo | grep processor | sort | uniq | wc -l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Get cpuinfo error: %v\n", err)
	}
	cpu_num, _ := strconv.Atoi(strings.TrimRight(out.String(), string(10)))
	//fmt.Println(cpu_num)
	for {
		GetMetrics(agentInfo, conf, 3, etcdclient)
		pubmessage := fmt.Sprintf("hm.agent.heartbeat.%s", conf.Agent_id)
		fmt.Println(pubmessage)
		for i := 0; i < len(agentInfo.Job); i++ {
			if agentInfo.Job[i].Type == SYSTEM {
				heartBeatInfo := SysHeartBeat{}
				heartBeatInfo.Address = agentInfo.Address
				heartBeatInfo.Index = 0
				heartBeatInfo.Job = agentInfo.Job[i].Name
				heartBeatInfo.Job_state = agentInfo.Job[i].Job_state
				heartBeatInfo.Ntp = agentInfo.Ntp
				heartBeatInfo.Traffic = agentInfo.Traffic
				heartBeatInfo.Vitals = agentInfo.Vitals
				heartBeatInfo.Cores = cpu_num
				data, err := json.Marshal(heartBeatInfo)
				if err != nil {
					fmt.Printf("Json Marshal agentInfo error : %v\n", err)
				}
				//fmt.Println(string(data))
				nats.NatsPub(pubmessage, nc, data)
			} else {
				heartBeatInfo := ProcessHeartBeat{}
				heartBeatInfo.Address = agentInfo.Address
				heartBeatInfo.Index = agentInfo.Job[i].Index
				heartBeatInfo.Job = agentInfo.Job[i].Name
				heartBeatInfo.Job_state = agentInfo.Job[i].Job_state
				heartBeatInfo.Ntp = agentInfo.Ntp
				heartBeatInfo.Vitals = agentInfo.Job[i].ProcessData
				data, err := json.Marshal(heartBeatInfo)
				if err != nil {
					fmt.Printf("Json Marshal agentInfo error : %v\n", err)
				}
				nats.NatsPub(pubmessage, nc, data)
			}
		}

		time.Sleep(time.Second * time.Duration(conf.Heartbeat_interval))
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
		//conf = config.GetTemplate()
		fmt.Printf("Error: No config file find in path: %s. \n", *config_path)
		return
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
		fmt.Println("We don't have a daemon here. Use agentserver.")
	}
	util.Trap(func() {})
}
