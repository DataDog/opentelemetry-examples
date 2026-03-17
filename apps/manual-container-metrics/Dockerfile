# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache 2.0 License.
#
FROM golang:1.24 AS builder

WORKDIR /app
COPY app manual-container-metrics-app/

WORKDIR /app/manual-container-metrics-app
RUN go mod download
RUN CGO_ENABLED=0 go build -o /manual-container-metrics-app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /manual-container-metrics-app /manual-container-metrics-app

EXPOSE 3000

CMD [ "/manual-container-metrics-app" ]
