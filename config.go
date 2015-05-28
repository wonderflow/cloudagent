package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Base_dir           string
	Logfile            string
	Loglevel           string
	Mbus               string
	Heartbeat_interval int
	Monit_files        string
	Monit_url          string
	Etcd_url           string
	Etcd_dir           string
	Job_name           string
	Job_index          int
	Agent_id           string
}

func GetConfig(config_file string) (*Config, error) {
	buf, err := ioutil.ReadFile(config_file)
	if err != nil {
		return nil, err
	}
	conf := Config{}
	err = yaml.Unmarshal(buf, &conf)
	return &conf, err
}
