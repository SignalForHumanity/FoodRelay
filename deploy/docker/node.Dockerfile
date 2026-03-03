FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/node ./cmd/node

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/node .
COPY migrations/ ./migrations/
EXPOSE 8080
ENTRYPOINT ["./node"]
