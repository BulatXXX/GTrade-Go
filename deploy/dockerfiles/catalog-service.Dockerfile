FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY shared/httpmiddleware ./shared/httpmiddleware
COPY services/catalog-service ./services/catalog-service

WORKDIR /src/services/catalog-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN adduser -D appuser
USER appuser
WORKDIR /home/appuser

COPY --from=builder /out/server ./server

EXPOSE 8084

CMD ["./server"]
