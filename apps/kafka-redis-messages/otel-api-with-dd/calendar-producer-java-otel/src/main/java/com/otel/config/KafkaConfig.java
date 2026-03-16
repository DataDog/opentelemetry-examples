package com.otel.config;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.instrumentation.kafkaclients.KafkaTelemetry;
import java.util.Properties;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.Producer;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.common.serialization.StringSerializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Kafka producer configuration with OTel API and DD backend.
 *
 * DD-specific: KafkaTelemetry.create(GlobalOpenTelemetry.get()) uses the DD-provided
 * OTel implementation when DD_TRACE_OTEL_ENABLED=true. The dd-java-agent provides
 * the GlobalOpenTelemetry instance.
 *
 * The wrapped producer injects W3C Trace Context headers into Kafka messages.
 * DD_TRACE_PROPAGATION_STYLE_INJECT=tracecontext must be set so the DD agent
 * uses W3C format (instead of DD's native x-datadog-* headers) for Kafka.
 */
@Configuration
public class KafkaConfig {
  private final Logger log = LoggerFactory.getLogger(KafkaConfig.class);

  @Value("${kafka.servers:localhost:9092}")
  private String bootstrapServers;

  @Bean
  public Producer<String, String> producer() {
    Properties properties = new Properties();
    log.info("using bootstrapServers:" + bootstrapServers);
    properties.setProperty(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
    properties.setProperty(
        ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
    properties.setProperty(
        ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());

    // DD-specific: GlobalOpenTelemetry.get() returns the DD-backed OTel instance.
    KafkaTelemetry telemetry = KafkaTelemetry.create(GlobalOpenTelemetry.get());
    Producer<String, String> tracingProducer =
        telemetry.wrap(new KafkaProducer<String, String>(properties));
    return tracingProducer;
  }
}
