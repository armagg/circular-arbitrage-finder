FROM golang:1.27.3-alpine3.24 AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY config.yaml ./

RUN CGO_ENABLED=0 GOOS=linux go build -o arb-finder ./cmd/arb-finder

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/arb-finder .
COPY --from=builder /app/config.yaml .

EXPOSE 50052

CMD ["./arb-finder"]
