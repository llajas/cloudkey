version := $(shell git describe --tags)
revision := $(shell git rev-parse HEAD)
release := $(shell git describe --tags | cut -d"-" -f 1,2)
build_date := $(shell date -u +"%Y-%m-%dT%H:%M:%S+00:00")
application := $(shell basename `pwd`)

GO_LDFLAGS := "-X github.com/jnovack/go-version.Application=${application} -X github.com/jnovack/go-version.Version=${version} -X github.com/jnovack/go-version.Revision=${revision} -X github.com/jnovack/go-version.BuildDate=${build_date}"

all: buildnew

.PHONY: install
install:
	sudo cp cloudkey.service /lib/systemd/system/cloudkey.service
	sudo cp cloudkey /usr/local/bin/cloudkey
	sudo systemctl daemon-reload

.PHONY: update
update:
	sudo systemctl stop cloudkey
	sudo cp cloudkey /usr/local/bin/cloudkey
	sudo systemctl start cloudkey

.PHONY: stop
stop:
	sudo systemctl stop cloudkey

.PHONY: build
build:
	sudo GOOS=linux GOARCH=arm64 go build -ldflags $(GO_LDFLAGS) cloudkey.go

.PHONY: buildnew
buildnew:
	sudo GOOS=linux GOARCH=arm64 go build cloudkey.go

.PHONY: clean
clean:
	sudo rm -f cloudkey

.PHONY: restart
restart: build
	sudo systemctl restart cloudkey

.PHONY: status
status:
	sudo systemctl status cloudkey

.PHONY: logs
logs:
	sudo journalctl -u cloudkey -f

.PHONY: test
test:
	./test_env_config.sh

.PHONY: quick-test
quick-test:
	./quick_test.sh
