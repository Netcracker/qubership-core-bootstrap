FROM golang:1.24 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership/core-base:1.0.0 AS run

ENV K8S_CORE_API_GROUP_NAMES=core.qubership.org K8S_CDN_API_GROUP_NAMES=cdn.qubership.org

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-checker

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-checker"]