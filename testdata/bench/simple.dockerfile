# Simple Dockerfile for benchmarking
FROM alpine:3.18
RUN apk add --no-cache curl
WORKDIR /app
CMD ["curl", "-s", "http://example.com"]
