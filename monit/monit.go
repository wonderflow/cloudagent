package monit

import (
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"encoding/xml"
	"fmt"
	"github.com/wonderflow/cloudagent/config"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Service struct {
	ServiceName string `xml:"name,attr"`
	//file_system => 0,directory => 1,file => 2,process => 3,remote_host => 4,system => 5,fifo => 6
	Type           int   `xml:"type"`
	Collected_sec  int64 `xml:"collected_sec"`
	Collected_usec int64 `xml:"collected_usec"`
	Status         int   `xml:"status"`
	Status_hint    int   `xml:"status_hint"`
	//MONITOR： no => 0,yes => 1,init => 2
	Monitor       int     `xml:"monitor"`
	Monitormode   int     `xml:"monitormode"`
	pendingaction int     `xml:"pendingaction"`
	Pid           int     `xml:"pid"`
	Ppid          int     `xml:"ppid"`
	Uptime        int     `xml:"uptime"`
	Children      int     `xml:"children"`
	Memory        Memory  `xml:"memory"`
	CpuPercent    float64 `xml:"cpu>percenttotal"`
	SysLoad       Load    `xml:"system>load"`
	SysCpu        Cpu     `xml:"system>cpu"`
	SysMemory     Memory  `xml:"system>memory"`
	SysSwap       Swap    `xml:"system>swap"`
}

type Load struct {
	Avg01 float64 `json:"avg01" xml:"avg01"`
	Avg05 float64 `json:"avg05" xml:"avg05"`
	Avg15 float64 `json:"avg15" xml:"avg15"`
}
type Cpu struct {
	User   float64 `json:"user" xml:"user"`
	System float64 `json:"sys" xml:"sys"`
	Wait   float64 `json:"wait" xml:"wait"`
}
type Memory struct {
	Percent float64 `json:"percent" xml:"percent"`
	Kb      int64   `json:"kb" xml:"kilobyte"`
}
type Swap struct {
	Percent float64 `json:"percent" xmL:"percent"`
	Kb      int64   `json:"kb" xml:"kilobyte"`
}

/*
example of system service

<service name="system_dawei-2-8c21b22d-cea5-4784-9384-be233a54b11c">
<type>5</type>
<collected_sec>1435294199</collected_sec>
<collected_usec>569413</collected_usec>
<status>0</status>
<status_hint>0</status_hint>
<monitor>1</monitor>
<monitormode>0</monitormode>
<pendingaction>0</pendingaction>
<system>
<load>
<avg01>0.06</avg01>
<avg05>0.03</avg05>
<avg15>0.05</avg15>
</load>
<cpu>
<user>0.1</user>
<system>0.1</system>
<wait>0.0</wait>
</cpu>
<memory>
<percent>3.2</percent>
<kilobyte>133472</kilobyte>
</memory>
<swap>
<percent>0.0</percent>
<kilobyte>0</kilobyte>
</swap>
</system>
</service>
</services>
*/

type PlatForm struct {
	Name    string `xml:"name"`
	Release string `xml:"release"`
	Machine string `xml:"machine"`
	CPU     int    `xml:"cpu"`
	Memory  int    `xml:"memory"`
	Swap    int    `xml:"swap"`
}
type MonitStatus struct {
	XMLName       xml.Name  `xml:"monit"`
	PlatFormValue PlatForm  `xml:"platform"`
	Services      []Service `xml:"services>service"`
}

func GetMonitStatusXML(conf *config.Config) io.Reader {
	var resp *http.Response
	buf, err := ioutil.ReadFile(conf.Monit_files)
	if err != nil {
		fmt.Printf("Read monit user file error : %v\n", err)
		return nil
	}
	userinfo := strings.Split(string(buf), ":")
	for i, s := range userinfo {
		userinfo[i] = strings.Trim(s, " ")
		userinfo[i] = strings.Trim(s, "\n")
	}
	url := conf.Monit_url

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://"+url+"/_status2?format=xml", nil)
	req.Header.Set("Content-Type", "application/xhtml+xml")
	req.SetBasicAuth(userinfo[0], userinfo[1])
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Request monit status error %s %v\n", url, err)
		return nil
	}
	if resp.StatusCode != 200 {
		fmt.Printf("Request Monit error: %v\n", resp.StatusCode)
		return nil
	}
	return resp.Body
}

func ParseXML(raw io.Reader) (MonitStatus, error) {
	v := MonitStatus{}

	decoder := xml.NewDecoder(raw)
	decoder.CharsetReader = charset.NewReader
	err := decoder.Decode(&v)
	if err != nil {
		fmt.Printf("Parse monit XML error: %v\n", err)

		return v, err
	}
	return v, nil
}

func GetMonitStatus(conf *config.Config) MonitStatus {
	raw := GetMonitStatusXML(conf)
	v, _ := ParseXML(raw)
	return v
}
