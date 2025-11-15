# Multi-stage Dockerfile
FROM golang:1.21-alpine as builder

RUN apk add --no-cache git
WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy sources and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \ 
    go build -ldflags "-s -w" -o /app/app main.go

# Final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/app /app/app
EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/app/app"]
