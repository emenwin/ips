FROM golang:1.23-alpine as builder

# Install make for the build step
RUN apk add --no-cache make

COPY . /building
WORKDIR /building
RUN make build

FROM alpine:3 AS alpine

RUN apk add --no-cache ca-certificates

ENV IPS_DIR=/app/data
RUN mkdir -p /app/data

WORKDIR /
COPY --from=builder /building/bin/ips /bin/ips
VOLUME ["/app/data"]
EXPOSE 6860
ENTRYPOINT ["/bin/ips", "server"]
