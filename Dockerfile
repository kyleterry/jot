FROM golang:1.10-alpine as builder
WORKDIR /go/src/github.com/kyleterry/jot
COPY . .
RUN apk --no-cache add make git
RUN go get -u github.com/cloudflare/gokey/cmd/gokey
RUN make

FROM alpine:3.4
RUN apk --no-cache add bash
COPY --from=builder /go/src/github.com/kyleterry/jot/bin/jot /usr/bin/jot
COPY --from=builder /go/bin/gokey /usr/bin/gokey
COPY --from=builder /go/src/github.com/kyleterry/jot/docker-entrypoint.sh /
VOLUME /etc/jot
VOLUME /var/lib/jot
EXPOSE 8095
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["jot"]