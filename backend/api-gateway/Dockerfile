FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod init stock-gateway 2>/dev/null || true
RUN go get github.com/gin-gonic/gin gorm.io/gorm gorm.io/driver/postgres
RUN go build -o gateway main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/gateway .
EXPOSE 8080
CMD ["./gateway"]