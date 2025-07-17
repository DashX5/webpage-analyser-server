
FROM golang:1.24-alpine AS builder

WORKDIR /build


RUN apk add --no-cache git


COPY go.mod go.sum ./
RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o webpage-analyzer cmd/server/main.go


FROM alpine:3.19

WORKDIR /app


RUN apk add --no-cache ca-certificates tzdata


COPY --from=builder /build/webpage-analyzer /app/
COPY --from=builder /build/web /app/web
COPY --from=builder /build/config-files /app/config-files


ENV GIN_MODE=debug


EXPOSE 8080


CMD ["./webpage-analyzer"] 