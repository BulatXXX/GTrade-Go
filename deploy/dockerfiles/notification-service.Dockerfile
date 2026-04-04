FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY shared/httpmiddleware ./shared/httpmiddleware
COPY services/notification-service ./services/notification-service

WORKDIR /src/services/notification-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN adduser -D appuser
USER appuser
WORKDIR /home/appuser

COPY --from=builder /out/server ./server

EXPOSE 8085

CMD ["./server"]
