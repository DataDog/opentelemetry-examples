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
 * Kafka producer configuration with OTel instrumentation.
 *
 * The KafkaProducer is wrapped with KafkaTelemetry to automatically inject
 * W3C Trace Context headers (traceparent, tracestate) into Kafka record headers.
 * This enables distributed tracing across Kafka - the Go consumer extracts
 * these headers to link its spans to the producer trace.
 *
 * OTel messaging semantic conventions:
 * - messaging.system = kafka
 * - messaging.destination.name = topic name
 * - messaging.operation = publish
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

    // OTel: Wrap the KafkaProducer with KafkaTelemetry to automatically
    // inject trace context into Kafka message headers.
    KafkaTelemetry telemetry = KafkaTelemetry.create(GlobalOpenTelemetry.get());
    Producer<String, String> tracingProducer =
        telemetry.wrap(new KafkaProducer<String, String>(properties));
    return tracingProducer;
  }
}
