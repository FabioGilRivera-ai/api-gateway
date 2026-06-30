FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api-gateway .

FROM scratch
COPY --from=builder /app/api-gateway /api-gateway
EXPOSE 8080
ENTRYPOINT ["/api-gateway"]
