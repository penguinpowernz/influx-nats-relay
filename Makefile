VERSION=$(shell git describe --tags|tr -d 'v')

build:
	go build -o bin/influx-nats-relay ./cmd/influx-nats-relay

pkg:
	mkdir -p dpkg/buster/usr/bin
	mkdir -p dpkg/wheezy/usr/bin
	cp bin/influx-nats-relay dpkg/buster/usr/bin
	cp bin/influx-nats-relay dpkg/wheezy/usr/bin
	IAN_DIR=dpkg/buster ian set -v ${VERSION}-1buster
	IAN_DIR=dpkg/buster ian pkg
	IAN_DIR=dpkg/wheezy ian set -v ${VERSION}-1wheezy
	IAN_DIR=dpkg/wheezy ian pkg