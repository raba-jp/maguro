FROM golang:1.10-alpine as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
COPY . /go/src/github.com/vivitInc/maguro
RUN apk add --update --virtual build-dependencies \
      git \
      make
WORKDIR /go/src/github.com/vivitInc/maguro
RUN make && \
       apk del build-dependencies

FROM scratch
COPY --from=builder /go/src/github.com/vivitInc/maguro/maguro /maguro
EXPOSE 3000
ENTRYPOINT ["/maguro"]
