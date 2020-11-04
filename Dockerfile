FROM golang:1.15-alpine as builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /go/src/cachenator
COPY *.go go.mod /go/src/cachenator/

RUN go mod download

RUN go build -o /go/bin/cachenator

# Runner
FROM alpine

LABEL org.opencontainers.image.source https://github.com/adrianchifor/cachenator

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=builder /go/bin/cachenator /

# Disable debug logs in Gin http server and listen over 0.0.0.0
ENV GIN_MODE release

ENTRYPOINT ["/cachenator"]
