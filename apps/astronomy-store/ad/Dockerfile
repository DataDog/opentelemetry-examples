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

## --- BEGIN Datadog Git/VCS env vars (DD_GIT_*) ---
## Comment out or delete this block if you don't want Datadog-native Source Code Integration tagging.
ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}
## --- END Datadog Git/VCS env vars (DD_GIT_*) ---

## --- BEGIN OpenTelemetry vcs.* resource attributes ---
## Read at container start by otel-env.sh, which adds them to OTEL_RESOURCE_ATTRIBUTES.
ENV VCS_REPOSITORY_URL_FULL=${VCS_REPOSITORY_URL_FULL}
ENV VCS_REF_HEAD_REVISION=${VCS_REF_HEAD_REVISION}
COPY --from=shared otel-env.sh /usr/local/bin/otel-env.sh
## --- END OpenTelemetry vcs.* resource attributes ---

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/otel-env.sh", "java", "-jar", "app.jar"]
