FROM gradle:8.14-jdk17 AS build

WORKDIR /src
COPY build.gradle settings.gradle ./
RUN gradle dependencies --no-daemon

COPY src ./src
RUN gradle installDist --no-daemon

FROM eclipse-temurin:17-jre

WORKDIR /app
COPY --from=build /src/build/install/java-forwarder ./

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION

ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}

ENV VCS_REPOSITORY_URL_FULL=${VCS_REPOSITORY_URL_FULL}
ENV VCS_REF_HEAD_REVISION=${VCS_REF_HEAD_REVISION}
COPY --from=shared otel-env.sh /usr/local/bin/otel-env.sh

ENTRYPOINT ["/usr/local/bin/otel-env.sh", "/app/bin/java-forwarder"]
