FROM golang:1.24 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership/core-base:1.0.0 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-synchronizer

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-synchronizer"]