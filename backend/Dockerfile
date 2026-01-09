FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target="/go/pkg/mod" \
    go build -o bin/backend cmd/app/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/backend .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./backend"]
