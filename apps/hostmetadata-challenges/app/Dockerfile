FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /trace-generator .

FROM alpine:3.19
COPY --from=builder /trace-generator /trace-generator
EXPOSE 8080
ENTRYPOINT ["/trace-generator"]
