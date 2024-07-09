FROM golang:latest AS builder
WORKDIR /app
COPY ./ ./
RUN go mod download
RUN CGO_ENABLED=0 go build -o ./main
EXPOSE 8080
ENTRYPOINT ["./main"]
