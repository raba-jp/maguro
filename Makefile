.PHONY: all

all: depend build

depend:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

run:
	go run *.go

build:
	go build -a -installsuffix cgo

clean:
	-rm maguro
