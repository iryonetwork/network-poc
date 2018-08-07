FROM golang:1.9-alpine

RUN apk add --no-cache gcc musl-dev git
RUN go get github.com/ethereum/go-ethereum
RUN go get github.com/caarlos0/env && \
	go get github.com/golang/protobuf/...
RUN go get google.golang.org/grpc && \
	go get golang.org/x/net/context
RUN go get github.com/eoscanada/eos-go && \
	go get github.com/gorilla/mux && \
	go get github.com/gorilla/websocket && \
	go get github.com/segmentio/ksuid && \
	go get github.com/lucasjones/reggen
	