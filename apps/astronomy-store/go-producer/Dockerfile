FROM golang:1.26 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go install github.com/DataDog/orchestrion@v1.15.0
RUN orchestrion pin
RUN CGO_ENABLED=0 orchestrion go build -o /go-producer .

FROM gcr.io/distroless/static-debian12

COPY --from=build /go-producer /go-producer

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION

ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}

ENTRYPOINT ["/go-producer"]
