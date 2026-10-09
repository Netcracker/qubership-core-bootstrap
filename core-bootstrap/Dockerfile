FROM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o platform-bootstrap-image .

FROM ghcr.io/netcracker/qubership-core-base:alpine-2.5.7@sha256:f0836415633bc2a63c2a0431c4c0701d17bd7b31132176b26651b49927d9ad7e AS run

COPY --chown=10001:0 --chmod=555 --from=build app/platform-bootstrap-image /app/platform-bootstrap-image

WORKDIR /app

USER 10001:10001

CMD ["/app/platform-bootstrap-image"]