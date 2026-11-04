FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:8.0 AS build

WORKDIR /src
COPY DotnetForwarder.csproj .
RUN dotnet restore

COPY Program.cs .
RUN dotnet publish --configuration Release --output /app/publish --no-restore

FROM mcr.microsoft.com/dotnet/runtime:8.0

WORKDIR /app
COPY --from=build /app/publish .

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION

ENV DD_GIT_REPOSITORY_URL=${VCS_REPOSITORY_URL_FULL}
ENV DD_GIT_COMMIT_SHA=${VCS_REF_HEAD_REVISION}

ENV VCS_REPOSITORY_URL_FULL=${VCS_REPOSITORY_URL_FULL}
ENV VCS_REF_HEAD_REVISION=${VCS_REF_HEAD_REVISION}
COPY --from=shared otel-env.sh /usr/local/bin/otel-env.sh

ENTRYPOINT ["/usr/local/bin/otel-env.sh", "dotnet", "DotnetForwarder.dll"]
