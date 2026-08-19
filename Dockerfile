FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway

# postgresql-client is here for the migration step Render runs as a
# preDeployCommand (render.yaml) — the app itself only needs the binary.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/gateway ./gateway
COPY migrations ./migrations
EXPOSE 8080
CMD ["./gateway"]
