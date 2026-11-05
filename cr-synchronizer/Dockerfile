FROM golang:1.27.3@sha256:a386c44c1c90025ad5bc66feea3fef23052bef3bace3609f7e3e10e1f9314f45 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o cr-synchronizer .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.10@sha256:51173b982248bb866b236f504579b71c0a9c4a41975b957595f9faa9ab3fb5d6 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/cr-synchronizer /app/cr-checker

WORKDIR /app

USER 10001:10001

CMD ["/app/cr-checker"]