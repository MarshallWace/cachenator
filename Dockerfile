FROM golang:1.15-alpine as builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /go/src/falcon
COPY . /go/src/falcon

RUN go mod download

RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/falcon

# Runner
FROM alpine

COPY --from=builder /go/bin/falcon /

# Disable debug logs in Gin http server and listen over 0.0.0.0
ENV GIN_MODE release

ENTRYPOINT ["/falcon"]
