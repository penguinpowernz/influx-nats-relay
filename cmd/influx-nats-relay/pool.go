package main

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nats-io/nats.go"
)

type pool struct {
	conns []*nats.Conn
}

func newPool(connstring string) *pool {
	pl := &pool{}
	for _, url := range strings.Split(connstring, ",") {
		nc, err := nats.Connect(url)
		if err == nats.ErrNoServers {
			go pl.connectForever(url)
			break
		}
		pl.conns = append(pl.conns, nc)
	}
	return pl
}

func (pl *pool) connectForever(url string) {
	for {
		expo := backoff.NewExponentialBackOff()

		var err error
		var nc *nats.Conn
		err = backoff.Retry(func() error {
			nc, err = nats.Connect(url)
			return err
		}, expo)

		if err == nats.ErrNoServers {
			continue
		}

		pl.conns = append(pl.conns, nc)

		break
	}
}

var ErrNoServers = errors.New("no servers available")

func (pl *pool) Publish(topic string, data []byte) error {
	if len(pl.conns) == 0 {
		return ErrNoServers
	}

	for {
		n := rand.Intn(len(pl.conns))
		c := pl.conns[n]
		if c.IsConnected() {
			return c.Publish(topic, data)
		}
		time.Sleep(time.Second / 10)
	}
}
