FROM golang:1.18.1

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

RUN go build -o /bin/action ./cmd/action/

ENTRYPOINT ["/bin/action"]