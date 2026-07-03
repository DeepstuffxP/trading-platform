FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o trading-platform .



FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/trading-platform .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./trading-platform"]
