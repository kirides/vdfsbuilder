FROM golang:1.18.1 AS builder

WORKDIR /workspace

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


FROM scratch

# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/action /bin/action

ENTRYPOINT ["/bin/action"]
