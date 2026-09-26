FROM golang:1.27.1@sha256:3680233e3204827fbdc66088528ae6d4b3d034f51d03a99d454f6de034888244 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o platform-bootstrap-image .

FROM ghcr.io/netcracker/qubership-core-base:2.5.5@sha256:1e207c43e907165f9c55838bb853465357a28bfd50799cf4aa66b959fe72a836 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/platform-bootstrap-image /app/platform-bootstrap-image

WORKDIR /app

USER 10001:10001

CMD ["/app/platform-bootstrap-image"]