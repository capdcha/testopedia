FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server ./cmd/server
RUN go build -o /worker ./cmd/worker

FROM alpine:latest

RUN apk add --no-cache sqlite ca-certificates

COPY --from=builder /server /server
COPY --from=builder /worker /worker
COPY schema.sql /schema.sql

EXPOSE 8080

ENTRYPOINT ["/server"]
