package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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

func GetTemplate() *Config {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return &Config{
		Base_dir:           cwd,
		Logfile:            cwd + "/log",
		Loglevel:           "DEBUG",
		Mbus:               "nats://nats:c1oudc0w@10.10.101.165:4222/",
		Heartbeat_interval: 30,
		Monit_files:        "/var/vcap/monit/monit.user",
		Monit_url:          "10.10.101.152:2822",
		Etcd_url:           "http://10.10.101.146:2379",
		Etcd_dir:           "cloud_agent",
		Agent_id:           "testid",
	}
}
