package main

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nats-io/nats.go"
)

var ErrNoServers = errors.New("no servers available")

type pool struct {
	conns []*nats.Conn
}

func newPool(connstring string) *pool {
	pl := &pool{}
	for _, url := range strings.Split(connstring, ",") {
		nc, err := nats.Connect(url)
		if err == nats.ErrNoServers {
			go pl.ConnectForever(url)
			break
		}
		pl.conns = append(pl.conns, nc)
	}
	return pl
}

// Stats will returns nats.Statistics objects for every connection
// in the pool, indexed by the connection URL
func (pl *pool) Stats() map[string]nats.Statistics {
	s := map[string]nats.Statistics{}
	for _, nc := range pl.conns {
		s[nc.Opts.Url] = nc.Stats()
	}
	return s
}

// ConnectForever will take the given URL and provided the error message
// represents a connection refused condition (nats.ErrNoServers) it will
// continue trying to connect to the URL using a backoff algorithm
func (pl *pool) ConnectForever(url string) {
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

// Publish will select a random connection from the pool and call the
// publish method on it to send the given data to the given topic
// in a rudimentary method of load balancing
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
