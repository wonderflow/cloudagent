package etcd

import (
	"errors"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/wonderflow/cloudagent/config"
	"log"
	"strings"
	"time"
)

func Connect(conf *config.Config, ip string) *etcd.Client {
	if strings.Contains(conf.Etcd_url, "http://") != true {
		conf.Etcd_url = "http://" + conf.Etcd_url
	}
	machines := []string{conf.Etcd_url}
	client := etcd.NewClient(machines)
	return client
}

func EtcdHup(client *etcd.Client, conf *config.Config, ip string) {
	go beat(client, ip, conf)
}

func beat(client *etcd.Client, ip string, conf *config.Config) {
	for {
		if _, err := client.Set(conf.Etcd_dir, ip, uint64(conf.Heartbeat_interval*2)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%v beat sent.\n", time.Now())
		time.Sleep(time.Second * time.Duration(conf.Heartbeat_interval))
	}
}

func GetIndex(client *etcd.Client, basedir string, jobname string, ip string) (int, error) {
	jobdir := "/jobs" + "/" + jobname
	response, err := client.Get(jobdir, true, true)
	if err == nil {
		for i := 0; i < response.Node.Nodes.Len(); i++ {
			if response.Node.Nodes[i].Value == ip {
				return i, nil
			}
		}
	}
	response, err = client.AddChild(jobdir, ip, 0)
	if err != nil {
		fmt.Printf("use etcd to get index error: %v\n", err)
		return 0, err
	}
	mykey := response.Node.Key
	response, err = client.Get(jobdir, true, true)
	if err != nil {
		fmt.Printf("get etcd jobdir error: %v\n", err)
		return 0, err
	}
	for i := 0; i < response.Node.Nodes.Len(); i++ {
		if response.Node.Nodes[i].Key == mykey {
			return i, nil
		}
	}
	// this line would never reach.
	return 0, errors.New("etcd add child error!")
}
