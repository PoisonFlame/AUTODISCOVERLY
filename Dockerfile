FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /autodiscoverly ./cmd/autodiscoverly

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /autodiscoverly /autodiscoverly

ENV CONFIG_PATH=/etc/autodiscoverly/config.yaml
EXPOSE 8080
USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD ["/autodiscoverly", "-healthcheck"]

ENTRYPOINT ["/autodiscoverly"]
