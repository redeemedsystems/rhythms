# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO disabled: modernc.org/sqlite is pure Go, so the binary is fully static
# and needs nothing from the base image at runtime beyond ca-certificates.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rhythms ./cmd/rhythms
# distroless/static has no shell to mkdir at runtime, so /data has to exist
# with the right ownership before it ever reaches that image — create it
# here, in a stage that has one, and COPY --chown it across below.
RUN mkdir -p /data

# distroless/static: no shell, no package manager, just enough libc-free
# runtime for a static Go binary plus ca-certificates and a built-in
# non-root "nonroot" user (uid 65532) — nothing else to attack.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rhythms /usr/local/bin/rhythms
COPY --from=build --chown=nonroot:nonroot /data /data

USER nonroot:nonroot
WORKDIR /data
VOLUME ["/data"]
ENV RHYTHMS_ADDR=:8080
ENV RHYTHMS_DB_PATH=/data/rhythms.db
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/rhythms"]
