FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o platform-bootstrap-image .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.15@sha256:a3af3751d9187ce7f8ffd67a5aa8465fee78056c79f964bfa7727ea00da681da AS run

COPY --chown=10001:0 --chmod=555 --from=build app/platform-bootstrap-image /app/platform-bootstrap-image

WORKDIR /app

USER 10001:10001

CMD ["/app/platform-bootstrap-image"]