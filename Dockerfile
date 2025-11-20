FROM ubuntu:24.04

WORKDIR /app

# Install Go and system dependencies
RUN apt update && apt install -y \
    wget \
    ca-certificates \
    && wget -q https://go.dev/dl/go1.23.3.linux-arm64.tar.gz \
    && tar -C /usr/local -xzf go1.23.3.linux-arm64.tar.gz \
    && rm go1.23.3.linux-arm64.tar.gz \
    && apt clean \
    && rm -rf /var/lib/apt/lists/*

# Set Go environment variables
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# WebKit environment variables for better compatibility in containers
ENV WEBKIT_DISABLE_DMABUF_RENDERER=1
ENV WEBKIT_DISABLE_COMPOSITING_MODE=1

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Install Playwright and WebKit browser with all dependencies
# The --with-deps flag will install all required system packages
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps webkit

# Refresh CA certificates to ensure all trusted certificates are available
RUN update-ca-certificates

# Download and install RapidSSL TLS RSA CA G1 intermediate certificate
# This is needed for brownrice.com certificates
RUN wget -q -O /usr/local/share/ca-certificates/rapidssl-tls-rsa-ca-g1.crt \
    https://cacerts.digicert.com/RapidSSLTLSRSACAG1.crt.pem && \
    chmod 644 /usr/local/share/ca-certificates/rapidssl-tls-rsa-ca-g1.crt && \
    update-ca-certificates

# Copy source code
COPY . .
# Explicitly copy fonts directory if it exists
COPY fonts/ /app/fonts/
RUN CGO_ENABLED=0 go build -o wd-worker ./cmd/wd-worker

# Create directories
RUN mkdir -p /app/assets /app/rendered

# Health check helper
RUN echo "#!/bin/sh\ntest -f /app/wd-worker && exit 0 || exit 1" > /healthcheck.sh && chmod +x /healthcheck.sh

CMD ["sh", "-c", "while true; do sleep 3600; done"]

