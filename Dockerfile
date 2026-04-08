FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o tvproxy-streams ./cmd/tvproxy-streams/

FROM alpine:3.19
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/tvproxy-streams /usr/local/bin/
EXPOSE 8090
ENTRYPOINT ["tvproxy-streams"]
