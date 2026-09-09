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

WORKDIR /app

COPY --from=build /app/worker .

ENTRYPOINT ["./worker"]