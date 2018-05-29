.PHONY: all run clean
SRCS    := $(shell find . -type f -name '*.go')

all: depend build

depend:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

run:
	go run *.go

build: $(SRCS)
	go build -a -installsuffix cgo

clean:
	-rm maguro
