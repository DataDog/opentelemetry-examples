FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY app.csproj ./
RUN dotnet restore
COPY Program.cs ./
COPY probes.json ./
RUN dotnet publish -c Release -o /app/out

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=build /app/out ./

# Datadog.Trace.Bundle ships the CLR profiler binaries under ./datadog/.
# Without these env vars + the createLogPath.sh setup, Tracer.Instance
# returns a stub that creates non-emitting spans, and OTEL_TRACES_EXPORTER=otlp
# has no effect. The CORECLR_PROFILER GUID is dd-trace-dotnet's public profiler
# clsid. CORECLR_PROFILER_PATH is resolved at startup so the same image works
# on linux/amd64 and linux/arm64.
RUN bash ./datadog/createLogPath.sh

COPY --from=build /src/probes.json /app/probes.json

ENV CORECLR_ENABLE_PROFILING=1
ENV CORECLR_PROFILER={846F5F1C-F9AE-4B07-969E-05C26BC060D8}
ENV DD_DOTNET_TRACER_HOME=/app/datadog

EXPOSE 5000
CMD ["bash", "-c", "case \"$(uname -m)\" in \
        x86_64)  ARCH_DIR=linux-x64 ;; \
        aarch64|arm64) ARCH_DIR=linux-arm64 ;; \
        *) echo \"Unsupported arch: $(uname -m)\"; exit 1 ;; \
     esac; \
     export CORECLR_PROFILER_PATH=/app/datadog/${ARCH_DIR}/Datadog.Trace.ClrProfiler.Native.so; \
     if [ -f /app/datadog/${ARCH_DIR}/Datadog.Linux.ApiWrapper.x64.so ]; then \
       export LD_PRELOAD=/app/datadog/${ARCH_DIR}/Datadog.Linux.ApiWrapper.x64.so; \
     fi; \
     exec dotnet app.dll"]
