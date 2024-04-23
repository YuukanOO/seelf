FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o app main.go

FROM alpine:3.16 AS runner
RUN addgroup --system app
RUN adduser --system app
USER app
WORKDIR /app
COPY --from=builder --chown=app:app /app/app ./
ENTRYPOINT ["./app"]