package nats

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats"
	"strings"
)

func Natstest(natsURL string) {
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(natsURL, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}
	nc, err := opts.Connect()
	if err != nil {
		fmt.Printf("Can't connect: %v\n", err)
	}
	//defer opts.Close()

	// Simple Publisher
	nc.Publish("help", []byte("Hello World"))

	// Simple Async Subscriber
	nc.Subscribe("help", func(msg *nats.Msg) {
		fmt.Printf("Received on [%s]: '%s'\n", msg.Subject, string(msg.Data))
	})

	// EncodedConn can Publish any raw Go type using the registered Encoder
	type person struct {
		Name    string
		Address string
		Age     int
	}

	// Go type Subscriber
	nc.Subscribe("hello", func(msg *nats.Msg) {
		p := &person{}
		json.Unmarshal(msg.Data, p)

		fmt.Printf("Received on [%s]: '%v'\n", msg.Subject, p)
	})

	me := &person{Name: "derek", Age: 22, Address: "585 Howard Street, San Francisco, CA"}

	// Go type Publisher
	mebyte, _ := json.Marshal(me)
	nc.Publish("hello", mebyte)

}

func NatsConnect(natsURL string) (*nats.Conn, error) {
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(natsURL, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}
	nc, err := opts.Connect()
	if err != nil {
		fmt.Printf("Can't connect: %v\n", err)
	}
	return nc, err
}

func NatsPub(subject string, nc *nats.Conn, data []byte) {
	nc.Publish(subject, data)
}
