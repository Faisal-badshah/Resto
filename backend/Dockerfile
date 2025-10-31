FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /restaurant-backend ./...

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /restaurant-backend /usr/local/bin/restaurant-backend
EXPOSE 8080
ENV PORT=8080
CMD ["/usr/local/bin/restaurant-backend"]
