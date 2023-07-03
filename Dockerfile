FROM golang:1.20-alpine AS builder
ADD . /lln
WORKDIR /lln
ARG version=unknown
ARG githash=unknown
ARG gomod=github.com/rkonfj/lln
RUN go build -ldflags "-s -w -X '$gomod/tools.Version=$version' -X '$gomod/tools.Commit=$githash'"

FROM alpine:3.18
WORKDIR /root
ADD config.yml /etc/lln.yml
COPY --from=builder /lln/lln /usr/bin/lln
ENTRYPOINT ["/usr/bin/lln"]
CMD ["-c", "/etc/lln.yml"]
