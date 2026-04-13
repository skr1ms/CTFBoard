FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    git \
    make \
    build-base \
    libwebp-dev

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ENV GOCACHE=/root/.cache/go-build
ENV CGO_ENABLED=1

RUN --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target="/go/pkg/mod" \
    go build -o bin/backend cmd/app/main.go && \
    go build -o bin/cleanup cmd/cleanup/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates libwebp

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /app/bin/backend .
COPY --from=builder /app/bin/cleanup /usr/local/bin/astroctfb-cleanup
COPY --from=builder /app/migrations ./migrations

RUN chown -R app:app /app

USER app

EXPOSE 8080

CMD ["./backend"]
