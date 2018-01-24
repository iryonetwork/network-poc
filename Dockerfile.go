FROM golang:1.9-alpine

RUN apk add --no-cache gcc musl-dev git
RUN go get github.com/ethereum/go-ethereum
RUN go get github.com/caarlos0/env && \
	go get github.com/golang/protobuf/...
RUN go get google.golang.org/grpc && \
	go get golang.org/x/net/context
