FROM golang:1.10-alpine as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
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
EXPOSE 3000
ENTRYPOINT ["/maguro"]
