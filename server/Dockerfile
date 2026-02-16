# server/Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o lumi-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/lumi-server .

ENV LUMI_ROOT=/notes
ENV LUMI_PORT=8080
ENV LUMI_PASSWORD=dev

EXPOSE 8080

CMD ["./lumi-server"]
