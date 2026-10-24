# Build stage runs natively on the build machine's architecture
# (--platform=$BUILDPLATFORM) and cross-compiles for the requested target via
# GOOS/GOARCH, so multi-arch builds don't pay for QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /autodiscoverly ./cmd/autodiscoverly

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /autodiscoverly /autodiscoverly

ENV CONFIG_PATH=/etc/autodiscoverly/config.yaml
EXPOSE 8080
USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/autodiscoverly", "-healthcheck"]

ENTRYPOINT ["/autodiscoverly"]
