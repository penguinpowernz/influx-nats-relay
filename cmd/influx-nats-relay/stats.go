package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"
)

func sendStats(statsURL string, pl *pool, svr *server) error {
	http.DefaultClient.Timeout = time.Second * 30

	var hostname string
	_hostname, err := os.Hostname()
	if _hostname != "" && err == nil {
		hostname = ",host=" + _hostname
	}

	pstats := pl.Stats()
	items := svr.hosts.Items()
	liveHosts := len(items)

	lines := bytes.NewBufferString("")

	line := "influx_nats_relay" + hostname
	line += fmt.Sprintf(" requests=%d", svr.reqCount)
	line += fmt.Sprintf(",upstream_errors=%d", svr.upstreamErrors)
	line += fmt.Sprintf(",goroutines=%d", runtime.NumGoroutine())
	line += fmt.Sprintf(",connected_sources=%d", liveHosts)
	line += fmt.Sprintf(",connected_targets=%d", len(pstats))
	line += fmt.Sprintf(" %d", time.Now().UnixNano())
	lines.WriteString(line + "\n")

	for url, stats := range pstats {
		line := "influx_nats_relay" + hostname + ",target=" + url
		line += " "
		line += fmt.Sprintf("in_bytes=%d", stats.InBytes)
		line += fmt.Sprintf(",in_msgs=%d", stats.InMsgs)
		line += fmt.Sprintf(",out_msgs=%d", stats.OutMsgs)
		line += fmt.Sprintf(",out_bytes=%d", stats.OutBytes)
		line += fmt.Sprintf(",reconnects=%d", stats.Reconnects)
		line += fmt.Sprintf(" %d", time.Now().UnixNano())
		lines.WriteString(line + "\n")
	}

	for name, count := range items {
		line := "influx_nats_relay" + hostname + ",ip=" + name
		line += fmt.Sprintf(" reqs=%d", count.Object)
		line += fmt.Sprintf(" %d", time.Now().UnixNano())
		lines.WriteString(line + "\n")
	}

	res, err := http.Post(statsURL, "application/octet-stream", lines)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusNoContent {
		data, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unexpected status: %d - %s", res.StatusCode, string(data))
	}

	return nil
}
