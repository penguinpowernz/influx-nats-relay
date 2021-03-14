# influx-nats-relay

A simple relay server for InfluxDB that mimics the `POST /write` endpoint and forwards data via NATS
rather than to an InfluxDB server.  This can be used to pass data through a service like benthos to
get rate limiting and batching, before sending to a real InfluxDB server or other services.

## Usage

The usage of the binary looks like so:

```
Usage of influx-nats-relay:
  -b string
        the address to serve the relay on (default ":9097")
  -j    send JSON packet instead of influx line protocol
  -s string
        the subject to use, $db and $precision not required for JSON mode (default "influx.raw.$db.$precision")
  -u string
        the NATS URL to connect to (default "nats://127.0.0.1:4222")
```

You choose what address to serve on, which NATS server to forward to, and the pattern for the NATS
subject. When not using JSON mode, the subject must contain `$db` and `$precision` as the NATS payload
will contain only raw influx line protocol, so any recievers would be missing that extra context.

When in JSON mode the subject does not require these fields but they will still be substituted if
present.  The JSON payload looks like so:

```
{
    "database": "mydatabase",
    "precision": "s",
    "data': "...influx line protocol..."
}
```

When calling the `POST /write` endpoint the database and precision must be specified with the query
params just like with a normal call to InfluxDB:

```
POST /write?db=ingen&precision=s

dinos,site=B,isla=sorna count=232,raptors=26,trex=5,gallimimus=77,brachiosaur=15 1615718855
dinos,site=A,isla=nublar count=62,raptors=5,trex=1,gallimimus=42,brachiosaur=5 1615718855
```