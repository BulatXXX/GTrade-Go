FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY shared/httpmiddleware ./shared/httpmiddleware
COPY services/user-asset-service ./services/user-asset-service

WORKDIR /src/services/user-asset-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN adduser -D appuser
USER appuser
WORKDIR /home/appuser

COPY --from=builder /out/server ./server

EXPOSE 8082

CMD ["./server"]
