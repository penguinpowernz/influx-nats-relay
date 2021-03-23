
build:
	go build -o bin/influx-nats-relay ./cmd/influx-nats-relay

pkg:
	mkdir -p dpkg-buster/usr/bin
	mkdir -p dpkg-wheezy/usr/bin
	cp bin/influx-nats-relay dpkg-buster/usr/bin
	cp bin/influx-nats-relay dpkg-wheezy/usr/bin
	IAN_DIR=dpkg-buster ian pkg
	IAN_DIR=dpkg-wheezy ian pkg