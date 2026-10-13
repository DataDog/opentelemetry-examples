FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build

WORKDIR /src
COPY ProductCatalog.csproj .
RUN dotnet restore

COPY . .
RUN dotnet publish --configuration Release --output /app/publish --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:8.0

WORKDIR /app
COPY --from=build /app/publish .

ENV ASPNETCORE_URLS=http://+:8080

ARG VCS_REPOSITORY_URL_FULL
ARG VCS_REF_HEAD_NAME
ARG VCS_REF_TYPE
ARG VCS_REF_HEAD_REVISION
ARG IMAGE_VERSION
ARG IMAGE_CREATED

## --- BEGIN OCI image labels (org.opencontainers.image.*) ---
## Record where the image comes from (https://specs.opencontainers.org/image-spec/annotations/). Setting
## title, version, and created also overrides labels inherited from the base image.
LABEL org.opencontainers.image.title="astronomy-store-product-catalog" \
      org.opencontainers.image.description="Astronomy Store product-catalog service" \
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

ENTRYPOINT ["dotnet", "ProductCatalog.dll"]
