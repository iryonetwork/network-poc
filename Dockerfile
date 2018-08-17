FROM golang:1.10.3-alpine

RUN apk add --no-cache gcc musl-dev git
RUN go get github.com/caarlos0/env && \
	go get github.com/golang/protobuf/...
RUN go get github.com/eoscanada/eos-go
RUN	go get github.com/gorilla/mux && \
	go get github.com/gorilla/websocket && \
	go get github.com/segmentio/ksuid && \
	go get github.com/lucasjones/reggen && \
	go get github.com/gofrs/uuid
	