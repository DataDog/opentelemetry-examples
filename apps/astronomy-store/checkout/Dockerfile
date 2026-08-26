FROM golang:1.25 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 go build -o /checkout .

FROM gcr.io/distroless/static-debian12

COPY --from=build /checkout /checkout

EXPOSE 8080

ENTRYPOINT ["/checkout"]
