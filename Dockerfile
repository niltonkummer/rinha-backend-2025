FROM golang:1.24-alpine AS builder

WORKDIR /app

ADD go.mod ./
RUN go mod download

ADD . ./
RUN apk add --no-cache dumb-init
# RUN go build -ldflags="-s -w" -o /bin/api /app/cmd/main.go
RUN go build -o /bin/api /app/cmd/main.go

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder ["/usr/bin/dumb-init", "/usr/bin/dumb-init"]
COPY --from=builder /bin/api /bin/api
ADD etc/config/server.env ./etc/config/server.env
EXPOSE 8080

ENTRYPOINT ["/usr/bin/dumb-init", "--"]