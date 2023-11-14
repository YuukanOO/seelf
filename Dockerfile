FROM node:18-alpine AS front_builder
WORKDIR /app
COPY ./cmd/serve/front/package*.json .
RUN npm ci
COPY ./cmd/serve/front .
RUN npm run build

FROM golang:1.21-alpine AS builder
# build-base needed to compile the sqlite3 dependency
RUN apk add --update-cache build-base
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=front_builder /app/build ./cmd/serve/front/build
RUN go build -ldflags="-s -w" -o seelf

FROM alpine:3.16
ENV DATA_PATH=/seelf/data
WORKDIR /app
COPY --from=builder /app/seelf ./
EXPOSE 8080
CMD ["./seelf", "-c", "/seelf/data/conf.yml", "serve"]