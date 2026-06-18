# Build from the repository root:
#   docker build -f cmd/jsonair/docker/Dockerfile -t jsonair .
#
# Run:
#   docker run -d \
#     -p 9191:9191 \
#     -e MYSQL_USERNAME=jsonair \
#     -e MYSQL_PASSWORD=secret \
#     -e MYSQL_HOST=db.internal \
#     -e MYSQL_PORT=3306 \
#     -e MYSQL_DATABASE=jsonair \
#     -e HTTP_LISTEN=:9191 \
#     -e HTTP_MODE=production \
#     -e JWT_TOKEN_SECRET=... \
#     -e JWT_TOKEN_EXPIRE=60 \
#     -e TOKEN_HMAC_SECRET=... \
#     -e CONFIG_ENCRYPT_SECRET=... \
#     jsonair

# ---- Stage 1: Build ----
FROM golang:1.26 AS builder

WORKDIR /src

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o /out/jsonair \
        ./cmd/jsonair

# ---- Stage 2: Runtime ----
# distroless/static includes CA certificates (needed for MYSQL_TLS=true)
# and runs with no shell, package manager, or other attack surface.
# wget is copied from busybox solely to support the HEALTHCHECK instruction.
FROM gcr.io/distroless/static-debian12

COPY --from=busybox:stable-musl /bin/wget /wget
COPY --from=builder /out/jsonair /jsonair

# RUNAS is required by the server at startup. The privilege drop itself is a
# no-op when the process is already running as a non-root user (UID != 0).
ENV RUNAS=nobody

USER nonroot:nonroot

EXPOSE 9191

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/wget", "-qO-", "http://localhost:9191/health"]

ENTRYPOINT ["/jsonair"]
