FROM golang:1.10

RUN mkdir -p /go/src/github.com/luizalabs/sindico
WORKDIR /go/src/github.com/luizalabs/sindico
COPY . /go/src/github.com/luizalabs/sindico

RUN make

ENTRYPOINT ["./sindico"]
