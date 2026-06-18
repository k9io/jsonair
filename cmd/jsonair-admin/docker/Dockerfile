# Build from the repository root:
#   docker build -f cmd/jsonair-admin/docker/Dockerfile -t jsonair-admin .
#
# Run:
#   docker run -d \
#     -p 8080:8080 \
#     -e ADMIN_USERNAME=admin \
#     -e ADMIN_PASSWORD=secret \
#     -e ADMIN_SESSION_SECRET=... \
#     -e CONFIG_ENCRYPT_SECRET=... \
#     -e MYSQL_USERNAME=jsonair \
#     -e MYSQL_PASSWORD=secret \
#     -e MYSQL_HOST=db.internal \
#     -e MYSQL_PORT=3306 \
#     -e MYSQL_DATABASE=jsonair \
#     -e HTTP_LISTEN=:8080 \
#     jsonair-admin

# ---- Stage 1: Build ----
FROM golang:1.26 AS builder

WORKDIR /src

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o /out/jsonair-admin \
        ./cmd/jsonair-admin

# ---- Stage 2: Runtime ----
# distroless/static includes CA certificates (needed for MYSQL_TLS=true)
# and runs with no shell, package manager, or other attack surface.
# wget is copied from busybox solely to support the HEALTHCHECK instruction.
FROM gcr.io/distroless/static-debian12

COPY --from=busybox:stable-musl /bin/wget /wget
COPY --from=builder /out/jsonair-admin /jsonair-admin

USER nonroot:nonroot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/wget", "-qO-", "http://localhost:8080/health"]

ENTRYPOINT ["/jsonair-admin"]
