ARG GO_BUILDER_IMAGE=golang:1.27.4@sha256:fe517acaa580cc7915457080f850c37e618b3c1e750f98b58fc6429b605147a1
ARG RUNTIME_IMAGE=ghcr.io/netcracker/qubership-core-base:alpine-2.5.13@sha256:891386ea91d90a1376e12f30b458fc23eb9793afd35f82817c90e4911cddd633
ARG RUNTIME_STAGE=public-runtime
FROM --platform=$BUILDPLATFORM ${GO_BUILDER_IMAGE} AS build

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ARG TARGETOS
ARG TARGETARCH
ARG SOURCE_DIR=.
ARG BOOTSTRAP_DIR=
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-p=2 GOMAXPROCS=2 GOMEMLIMIT=1GiB

WORKDIR /src/core-bootstrap
COPY ${SOURCE_DIR}/ .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=core-bootstrap-go-build,target=/root/.cache/go-build \
    mkdir -p /reports && \
    go test -mod=readonly -v -count=1 -timeout=15m -coverprofile=/reports/coverage.out ./... 2>&1 | tee /reports/unit-tests.log

RUN --mount=type=bind,target=/build-context \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=core-bootstrap-go-build,target=/root/.cache/go-build \
    set -eu; \
    if [[ -n "$BOOTSTRAP_DIR" ]]; then \
        cp -a "/build-context/$BOOTSTRAP_DIR" /src/private; \
        cd /src/private; \
        go test -mod=readonly -v -count=1 -timeout=15m -coverprofile=/reports/bootstrap-coverage.out ./... 2>&1 | tee /reports/bootstrap-tests.log; \
    fi; \
    mkdir -p /out; \
    GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -mod=readonly -o /out/bootstrap .; \
    go version -m /out/bootstrap > /reports/build-info.txt

FROM scratch AS artifact
COPY --from=build /reports/ /test-results/

FROM ${RUNTIME_IMAGE} AS public-runtime
COPY --chown=10001:0 --chmod=555 --from=build /out/bootstrap /app/platform-bootstrap-image
WORKDIR /app
USER 10001:10001
CMD ["/app/platform-bootstrap-image"]

FROM ${RUNTIME_IMAGE} AS private-runtime
COPY --chmod=555 --from=build /out/bootstrap /app/core-bootstrap-image
WORKDIR /app
USER 10001
CMD ["/app/core-bootstrap-image"]

FROM ${RUNTIME_STAGE} AS run
