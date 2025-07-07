FROM golang:alpine AS builder

RUN adduser -D -g '' appuser
RUN mkdir /app

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o app .

RUN chmod 0555 app

# FROM scratch
FROM debian:bullseye-slim

RUN apt update && apt install -y bash coreutils
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /app

COPY --from=builder /etc/passwd /etc/passwd

USER appuser

COPY --from=builder /app/app ./app_binary
COPY --from=builder /app/pkg ./pkg

EXPOSE 3000

CMD [ "./app_binary" ]