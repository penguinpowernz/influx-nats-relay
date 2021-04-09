package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	cache "github.com/patrickmn/go-cache"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func main() {
	var natsURL, bind, subj, statsURL string
	var jsonMode bool
	var statsInterval int

	flag.IntVar(&statsInterval, "i", 0, "how often to write the stats (0 = disabled)")
	flag.StringVar(&statsURL, "s", "http://localhost:8186/write", "the InfluxDB server/telegraf listener to write stats to")
	flag.StringVar(&natsURL, "u", nats.DefaultURL, "the NATS URL to connect to")
	flag.StringVar(&bind, "b", ":9097", "the address to serve the relay on")
	flag.StringVar(&subj, "s", "influx.raw.$db.$precision", "the subject to use, $db and $precision not required for JSON mode")
	flag.BoolVar(&jsonMode, "j", false, "send JSON packet instead of influx line protocol")
	flag.Parse()

	haveAllFields := strings.Contains(subj, "$db") && strings.Contains(subj, "$precision")
	if !jsonMode && !haveAllFields {
		log.Println("must specify $db and $precision in the subject pattern")
		os.Exit(1)
	}

	pool := newPool(natsURL)

	dh := newRawDataHandler(pool.Publish, subj)
	if jsonMode {
		dh = newJSONDataHandler(pool.Publish, subj)
	}

	hostsCache := cache.New(5*time.Minute, 10*time.Minute)

	svr := &server{
		dataHandler: dh,
		hosts:       hostsCache,
	}

	// log the stats every interval if enabled
	if statsInterval > 0 {
		go func() {
			t := time.NewTicker(time.Second * time.Duration(statsInterval))
			for {
				<-t.C
				sendStats(statsURL, pool, svr)
			}
		}()
	}

	api := gin.Default()
	api.POST("/write", svr.httpHandler)
	api.Run(bind)
}

type server struct {
	dataHandler dataHandler
	hosts       *cache.Cache

	reqCount       int64
	upstreamErrors int64
}

func (svr *server) httpHandler(c *gin.Context) {
	db := c.Query("db")
	pr := c.Query("precision")

	if db == "" {
		c.AbortWithStatusJSON(400, map[string]string{"message": "must specify database"})
		return
	}

	if pr == "" {
		c.AbortWithStatusJSON(400, map[string]string{"message": "must specify precision"})
		return
	}

	defer c.Request.Body.Close()
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(400, "failed to read payload body: "+err.Error())
		log.Printf("ERROR: %s: %s", c.ClientIP(), err)
		return
	}

	if len(data) == 0 {
		c.AbortWithStatusJSON(400, "no data given in payload")
		log.Printf("ERROR: %s: %s", c.ClientIP(), "no data given in payload")
		return
	}

	svr.hosts.IncrementInt(c.ClientIP(), 1)
	svr.reqCount++

	if err := svr.dataHandler(db, pr, data); err != nil {
		c.AbortWithStatusJSON(504, "upstream server returned error: "+err.Error())
		log.Println("ERROR:", err)
		svr.upstreamErrors++
		return
	}

	c.Status(204)
}

type dataHandler func(string, string, []byte) error
type publishFunc func(string, []byte) error

func newRawDataHandler(publish publishFunc, ptn string) dataHandler {
	return func(db string, precision string, data []byte) error {
		ptn = strings.Replace(ptn, "$db", db, 1)
		ptn = strings.Replace(ptn, "$precision", precision, 1)
		return publish(ptn, data)
	}
}

func newJSONDataHandler(publish publishFunc, ptn string) dataHandler {
	return func(db string, precision string, data []byte) error {
		ptn = strings.Replace(ptn, "$db", db, 1)
		ptn = strings.Replace(ptn, "$precision", precision, 1)

		data, err := json.Marshal(map[string]string{
			"precision": precision,
			"database":  db,
			"data":      string(data),
		})

		if err != nil {
			return err
		}

		return publish(ptn, data)
	}
}
