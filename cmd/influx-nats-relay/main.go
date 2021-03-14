package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func main() {
	var natsURL, bind, subj string
	var jsonMode bool

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

	nc, err := nats.Connect(natsURL)
	if err != nil {
		panic(err)
	}

	dh := newRawDataHandler(nc, subj)
	if jsonMode {
		dh = newJSONDataHandler(nc, subj)
	}

	svr := &server{dh}
	api := gin.Default()

	api.POST("/write", svr.httpHandler)
	api.Run(bind)
}

type server struct {
	dataHandler dataHandler
}

func (svr *server) httpHandler(c *gin.Context) {
	db := c.Query("database")
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
		log.Println("ERROR:", err)
		return
	}

	if len(data) == 0 {
		c.AbortWithStatusJSON(400, "no data given in payload")
		return
	}

	if err := svr.dataHandler(db, pr, data); err != nil {
		c.AbortWithStatusJSON(504, "upstream server returned error: "+err.Error())
		log.Println("ERROR:", err)
		return
	}
}

type dataHandler func(string, string, []byte) error

func newRawDataHandler(nc *nats.Conn, ptn string) dataHandler {
	return func(db string, precision string, data []byte) error {
		ptn = strings.Replace(ptn, "$db", db, 1)
		ptn = strings.Replace(ptn, "$precision", precision, 1)
		return nc.Publish(ptn, data)
	}
}

func newJSONDataHandler(nc *nats.Conn, ptn string) dataHandler {
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

		return nc.Publish(ptn, data)
	}
}
