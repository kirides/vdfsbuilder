FROM golang:1.18.1 AS builder

ENV GO111MODULE=on

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

RUN go build \
  -ldflags "-s -w -extldflags '-static'" \
  -installsuffix cgo \
  -tags netgo \
  -o /bin/action \
  ./cmd/action/

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd



FROM scratch

# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc_passwd /etc/passwd
COPY --from=builder --chown=65534:0 /bin/action /bin/action

USER nobody

ENTRYPOINT ["/bin/action"]
