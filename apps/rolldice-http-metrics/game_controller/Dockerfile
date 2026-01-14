# Use OpenJDK 17 as the base image
FROM openjdk:17-jdk-slim

# Set the working directory
WORKDIR /game_controller

# Copy Gradle wrapper and build files
COPY gradlew .
COPY gradle gradle
COPY build.gradle .
COPY settings.gradle .

# Copy source code
COPY src src

# Make gradlew executable
RUN chmod +x ./gradlew

# Build the application
RUN ./gradlew build -x test

# Install curl and download OpenTelemetry Java agent
RUN apt-get update && apt-get install -y curl && \
    curl -L -o opentelemetry-javaagent.jar \
    https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar && \
    rm -rf /var/lib/apt/lists/*

ENV service_name=controller
ENV logs_exporter=otlp

# Expose port 5002 for the app to be accessible
EXPOSE 5002

# Define the command to run the app with OpenTelemetry agent
CMD ["java", "-javaagent:opentelemetry-javaagent.jar", "-jar", "build/libs/game_controller-0.0.1-SNAPSHOT.jar"]