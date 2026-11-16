FROM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.13@sha256:891386ea91d90a1376e12f30b458fc23eb9793afd35f82817c90e4911cddd633 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-checker

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-checker"]