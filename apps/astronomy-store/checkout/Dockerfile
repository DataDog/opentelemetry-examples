# Go 1.26: required by the dd-trace-go version that `orchestrion pin` resolves. Debian 12 (bookworm), as the
# runtime image, for a matching glibc.
FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go install github.com/DataDog/orchestrion@v1.13.1
RUN orchestrion pin
# CGO_ENABLED=1: dd-trace-go App & API Protection links libddwaf against glibc, it cannot find libdl in a
# distroless image otherwise.
RUN CGO_ENABLED=1 orchestrion go build -o /checkout .

# distroless/base rather than distroless/static: the cgo binary needs glibc.
FROM gcr.io/distroless/base-debian12

COPY --from=build /checkout /checkout

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION
ARG IMAGE_VERSION
ARG IMAGE_CREATED

## --- BEGIN OCI image labels (org.opencontainers.image.*) ---
## Record where the image comes from (https://specs.opencontainers.org/image-spec/annotations/). Setting
## title, version, and created also overrides labels inherited from the base image.
LABEL org.opencontainers.image.title="astronomy-store-checkout" \
      org.opencontainers.image.description="Astronomy Store checkout service" \
      org.opencontainers.image.source="${VCS_REPOSITORY_URL_FULL}" \
      org.opencontainers.image.revision="${VCS_REF_HEAD_REVISION}" \
      org.opencontainers.image.version="${IMAGE_VERSION}" \
      org.opencontainers.image.created="${IMAGE_CREATED}"
## --- END OCI image labels (org.opencontainers.image.*) ---

## --- BEGIN Datadog Git/VCS env vars (DD_GIT_*) ---
## Comment out or delete this block if you don't want Datadog-native Source Code Integration tagging.
ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}
## --- END Datadog Git/VCS env vars (DD_GIT_*) ---

EXPOSE 8080

ENTRYPOINT ["/checkout"]
