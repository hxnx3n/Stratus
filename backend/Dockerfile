FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o stratus .

FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/stratus .

RUN mkdir -p /data/storage

EXPOSE 8080

CMD ["./stratus"]
