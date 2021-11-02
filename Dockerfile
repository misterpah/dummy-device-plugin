FROM golang:1.16 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go

RUN go build -mod=readonly main.go

FROM centos:7

COPY --from=builder /workspace/main /usr/bin/sample-device-plugin

ENTRYPOINT ["/usr/bin/sample-device-plugin"]
