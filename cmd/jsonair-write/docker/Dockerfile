# Build from the repository root:
#   docker build -f cmd/jsonair-write/docker/Dockerfile -t jsonair-write .
#
# jsonair-write is the write-only API.  Do NOT publish its port on a public
# interface; bind it to an internal network and, ideally, require client
# certificates (HTTP_TLS=true and HTTP_CLIENT_CA).
#
# Run:
#   docker run -d \
#     -p 127.0.0.1:9192:9192 \
#     -e MYSQL_USERNAME=jsonair_write \
#     -e MYSQL_PASSWORD=secret \
#     -e MYSQL_HOST=db.internal \
#     -e MYSQL_PORT=3306 \
#     -e MYSQL_DATABASE=jsonair \
#     -e HTTP_LISTEN=:9192 \
#     -e HTTP_MODE=production \
#     -e WRITE_JWT_TOKEN_SECRET=... \
#     -e WRITE_JWT_TOKEN_EXPIRE=15 \
#     -e WRITE_TOKEN_HMAC_SECRET=... \
#     -e CONFIG_ENCRYPT_SECRET=... \
#     jsonair-write
#
# WRITE_JWT_TOKEN_SECRET, WRITE_TOKEN_HMAC_SECRET and CONFIG_ENCRYPT_SECRET must
# be three different values.  CONFIG_ENCRYPT_SECRET must match the one used by
# jsonair (the read API), otherwise it cannot decrypt what is written here.

# ---- Stage 1: Build ----
FROM golang:1.26 AS builder

WORKDIR /src

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o /out/jsonair-write \
        ./cmd/jsonair-write

# ---- Stage 2: Runtime ----
# distroless/static includes CA certificates (needed for MYSQL_TLS=true)
# and runs with no shell, package manager, or other attack surface.
# wget is copied from busybox solely to support the HEALTHCHECK instruction.
FROM gcr.io/distroless/static-debian12

COPY --from=busybox:stable-musl /bin/wget /wget
COPY --from=builder /out/jsonair-write /jsonair-write

# RUNAS is required at startup. The privilege drop itself is a no-op when the
# process is already running as a non-root user (UID != 0).
ENV RUNAS=nobody

USER nonroot:nonroot

EXPOSE 9192

# Note: if HTTP_TLS=true this plain-HTTP check will fail; override it with a
# check appropriate to your certificate setup.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/wget", "-qO-", "http://localhost:9192/health"]

ENTRYPOINT ["/jsonair-write"]
