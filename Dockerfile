# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26 AS build
WORKDIR /src

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags "-s -w" \
    -o /out/unifi-ha-presence-mqtt ./cmd

# --- runtime stage ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY --from=build /out/unifi-ha-presence-mqtt /unifi-ha-presence-mqtt

# Prod default: target a real UniFi controller. The health/probe port.
ENV MODE=live
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/unifi-ha-presence-mqtt"]
CMD ["serve"]
