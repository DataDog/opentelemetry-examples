# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache 2.0 License.
#

FROM golang:1.21.3
WORKDIR /app
COPY . .

RUN go mod tidy

RUN go build -o main

ENTRYPOINT ["/app/main"]
