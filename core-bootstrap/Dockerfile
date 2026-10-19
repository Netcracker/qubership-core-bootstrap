FROM golang:1.27.1@sha256:162be5298a40ed317005c8339c6de4d10d3eef336d66dc8e9259b03ab9d3a6d2 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o platform-bootstrap-image .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.8@sha256:0c2c9e4b49e73efffb30edf8d3de8b3eb43194a6c4f688da08ec3236b570b486 AS run

COPY --chown=10001:0 --chmod=555 --from=build app/platform-bootstrap-image /app/platform-bootstrap-image

WORKDIR /app

USER 10001:10001

CMD ["/app/platform-bootstrap-image"]