package main

import (
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Service struct {
	ServiceName    string  `xml:"name,attr"`
	Type           int     `xml:"type"`
	Collected_sec  int64   `xml:"collected_sec"`
	Collected_usec int64   `xml:"collected_usec"`
	Status         int     `xml:"status"`
	Status_hint    int     `xml:"status_hint"`
	Monitor        int     `xml:"monitor"`
	Monitormode    int     `xml:"monitormode"`
	pendingaction  int     `xml:"pendingaction"`
	Pid            int     `xml:"pid"`
	Ppid           int     `xml:"ppid"`
	Uptime         int     `xml:"uptime"`
	Children       int     `xml:"children"`
	MemPercent     float64 `xml:"memory>percenttotal"`
	CpuPercent     float64 `xml:"cpu>percenttotal"`
}

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

func GetMonitStatusXML(conf *Config) io.Reader {
	var resp *http.Response
	buf, err := ioutil.ReadFile(conf.Monit_files)
	if err != nil {
		fmt.Printf("Read monit user file error : %v\n", err)
		return nil
	}
	userinfo := strings.Split(string(buf), ":")
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
	return resp.Body
}

func ParseXML(raw io.Reader) (MonitStatus, error) {
	v := MonitStatus{}
	decoder := xml.NewDecoder(raw)
	decoder.CharsetReader = charset.NewReader
	err := decoder.Decode(&v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return v, err
	}
	fmt.Println(v)
	return v, nil
}

func GetMonitStatus(conf *Config) MonitStatus {
	raw := GetMonitStatusXML(conf)
	v, _ := ParseXML(raw)
	return v
}
