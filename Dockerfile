FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/invoice_generator ./main.go

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/invoice_generator ./invoice_generator
COPY --from=builder /src/templates ./templates
COPY --from=builder /src/static ./static

RUN mkdir -p /app/uploads

ENV GIN_MODE=release
EXPOSE 8080

CMD ["./invoice_generator"]
