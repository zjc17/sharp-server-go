FROM golang:1.21.1-alpine3.17 AS base

RUN apk update
RUN apk add --update --no-cache  \
    --repository https://dl-3.alpinelinux.org/alpine/edge/community  \
    --repository https://dl-3.alpinelinux.org/alpine/edge/main  \
    vips-dev

# Install dependencies only when needed
FROM base AS deps

#ENV GOPROXY=https://goproxy.cn

WORKDIR /app
# Install dependencies based on package manager
COPY go.mod go.sum ./
RUN go mod download

# Rebuild the source code only when needed
FROM base AS builder

RUN go install github.com/google/gops@latest

WORKDIR /app
COPY --from=deps /go/pkg/ /go/pkg/
COPY . .

RUN go build -o main main.go

# Production image, copy all the files and run
FROM alpine:3.17

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /go/bin/gops /usr/local/bin/
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./main"]