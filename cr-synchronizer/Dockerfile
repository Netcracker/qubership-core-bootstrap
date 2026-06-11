FROM golang:1.26@sha256:d47ca13cd596f3a338c1be5f79af628f42bedcf89455266211a9ab4f95da2828 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership-core-base:2.3.0@sha256:3ef9a4b348dcf26d1e9f63d375209c9b4b2359e0080fbcab1a566f6f6291b789 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-checker

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-checker"]