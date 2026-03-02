FROM golang:1.25 AS builder

WORKDIR /build

# Clone CoreDNS first
# Use stable version (v1.14.1) to avoid unstable branch surprises.
# We do this BEFORE copying local source so that changes to local source don't invalidate the clone.
RUN git clone --depth 1 --branch v1.14.1 https://github.com/coredns/coredns.git

# Copy source code of your project (plugins)
COPY . compass

# Register plugins in plugin.cfg
# Order matters! Insert them after 'metadata' so iplookup and steering
# run at the correct place in the middleware chain.
WORKDIR /build/coredns
RUN sed -i '/^metadata:metadata/a iplookup:compass/plugin/iplookup\nsteering:compass/plugin/steering' plugin.cfg

# Replace compass module with local copy
RUN go mod edit -replace compass=/build/compass

# Update dependencies and generate plugin registration code
# Run go generate first to update zplugin.go with new imports
RUN go generate
# Then get compass and tidy up
RUN go get compass
RUN go mod tidy

# Build static binary
RUN CGO_ENABLED=0 go build -o coredns

FROM scratch

# Copy root certificates (useful for HTTPS requests if needed)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder container
COPY --from=builder /build/coredns/coredns /usr/bin/coredns

# Default DNS ports
EXPOSE 53 53/udp

ENTRYPOINT ["/usr/bin/coredns"]
