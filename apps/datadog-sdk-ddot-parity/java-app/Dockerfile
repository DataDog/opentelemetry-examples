FROM eclipse-temurin:21-jdk

WORKDIR /app

ARG GIT_COMMIT=unknown
ENV OTEL_RESOURCE_ATTRIBUTES="deployment.environment.name=validation,service.version=dd-trace-java-1.66.0-run1,git.commit.sha=${GIT_COMMIT}"

ADD https://repo1.maven.org/maven2/com/datadoghq/dd-java-agent/1.66.0/dd-java-agent-1.66.0.jar /app/dd-java-agent.jar
ADD https://repo1.maven.org/maven2/io/opentelemetry/opentelemetry-api/1.54.0/opentelemetry-api-1.54.0.jar /app/lib/opentelemetry-api.jar
ADD https://repo1.maven.org/maven2/io/opentelemetry/opentelemetry-context/1.54.0/opentelemetry-context-1.54.0.jar /app/lib/opentelemetry-context.jar
ADD https://repo1.maven.org/maven2/io/opentelemetry/opentelemetry-common/1.54.0/opentelemetry-common-1.54.0.jar /app/lib/opentelemetry-common.jar
ADD https://repo1.maven.org/maven2/org/postgresql/postgresql/42.7.7/postgresql-42.7.7.jar /app/lib/postgresql.jar

COPY App.java ProbeTargets.java probes.json /app/
RUN javac -g -parameters -cp "/app/lib/*" -d /app/classes /app/App.java /app/ProbeTargets.java

EXPOSE 5000
CMD ["java", "-javaagent:/app/dd-java-agent.jar", "-cp", "/app/classes:/app/lib/*", "App"]
