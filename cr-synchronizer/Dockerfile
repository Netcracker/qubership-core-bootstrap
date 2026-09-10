FROM golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership-core-base:2.3.7@sha256:b917b3a1731a2ae26b507d22565f030ec25ff8d28b75a80b8b08bbc946f4d73b AS run

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-checker

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-checker"]