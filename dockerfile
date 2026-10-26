# Build with:
# docker build -t load-tester-worker:test .
# The image name must match the WorkerImage passed to NewDocker in the orchestrator.

FROM golang:1.26.5 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o worker ./cmd/worker

FROM debian:bookworm-slim

# add general CA Certs file to container
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists*
RUN curl -o /usr/local/share/ca-certificates/mozilla-cacert.crt https://curl.se
RUN update-ca-certificates

WORKDIR /app

COPY --from=build /app/worker .

ENTRYPOINT ["./worker"]