# base go image
FROM golang:1.22.3-alpine as builder

RUN mkdir /app

COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 go build -o bot ./cmd/app/main.go


RUN chmod +x /app/bot

# build a tiny docker image
FROM alpine:latest

RUN apk update && apk add --no-cache tzdata

ENV TZDIR=/usr/share/zoneinfo

RUN mkdir /app

WORKDIR /app

COPY --from=builder /app/bot /app

CMD [ "/app/bot" ]
