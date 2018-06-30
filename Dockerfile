FROM golang:1.10-alpine as builder
WORKDIR /go/src/github.com/kyleterry/jot
COPY . .
RUN apk --no-cache add make
RUN make build-go

FROM alpine:3.4
COPY --from=builder /go/src/github.com/kyleterry/jot/bin/jot /usr/bin/jot
ENTRYPOINT ["jot"]
