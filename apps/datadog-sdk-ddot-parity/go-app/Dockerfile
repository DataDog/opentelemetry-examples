FROM golang:1.25

WORKDIR /app
COPY go.mod main.go ./
RUN go mod tidy && go build -o /app/server .

EXPOSE 5000
CMD ["/app/server"]
