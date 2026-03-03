FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/registry ./cmd/registry

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/registry .
COPY migrations/ ./migrations/
EXPOSE 8081
ENTRYPOINT ["./registry"]
