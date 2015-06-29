package etcd

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/wonderflow/cloudagent/config"
	"log"
	"time"
)

func Connect(conf *config.Config, ip string) {
	machines := []string{conf.Etcd_url}
	client := etcd.NewClient(machines)

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
