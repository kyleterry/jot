FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags "-w -s" -o /bin/jot ./cmd/jot
RUN go build -ldflags "-w -s" -o /bin/gokey github.com/cloudflare/gokey/cmd/gokey

FROM alpine:3.21

LABEL org.opencontainers.image.source=https://github.com/kyleterry/jot

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/jot /bin/jot
COPY --from=builder /bin/gokey /bin/gokey
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

ENV JOT_SEED_FILE=/etc/jot/seed
ENV JOT_DATA_DIR=/var/lib/jot

VOLUME /etc/jot
VOLUME /var/lib/jot

EXPOSE 8095

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["jot"]
