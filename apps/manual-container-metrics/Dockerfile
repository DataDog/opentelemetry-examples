# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache 2.0 License.
#
FROM golang:1.20

WORKDIR /app
COPY app manual-container-metrics-app/

WORKDIR /app/manual-container-metrics-app
RUN go mod download

RUN go build -o /manual-container-metrics-app

CMD [ "/manual-container-metrics-app" ]
