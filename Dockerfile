FROM golang:1.10-alpine as builder

ENV CGO_ENABLED=0 \
      GOOS=linux \
      GOARCH=amd64
RUN apk add --update --no-cache \
      git \
      make && \
      go get -u github.com/golang/dep/cmd/dep
COPY . /go/src/github.com/vivitInc/maguro
WORKDIR /go/src/github.com/vivitInc/maguro
RUN make

FROM alpine:3.7
RUN apk add --update --no-cache ca-certificates
COPY --from=builder /go/src/github.com/vivitInc/maguro/maguro /maguro
COPY --from=builder /go/src/github.com/vivitInc/maguro/public /public
EXPOSE 3000
WORKDIR /
ENTRYPOINT ["/maguro"]
