FROM gradle:8.14-jdk17 AS build

WORKDIR /src
COPY build.gradle settings.gradle ./
RUN gradle dependencies --no-daemon

COPY src ./src
RUN gradle bootJar --no-daemon

FROM eclipse-temurin:17-jre

WORKDIR /app
COPY --from=build /src/build/libs/ad-0.0.1-SNAPSHOT.jar app.jar

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION
ARG IMAGE_VERSION
ARG IMAGE_CREATED

## --- BEGIN OCI image labels (org.opencontainers.image.*) ---
## Record where the image comes from (https://specs.opencontainers.org/image-spec/annotations/). Setting
## title, version, and created also overrides labels inherited from the base image.
LABEL org.opencontainers.image.title="astronomy-store-ad" \
      org.opencontainers.image.description="Astronomy Store ad service" \
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

ENTRYPOINT ["java", "-jar", "app.jar"]
