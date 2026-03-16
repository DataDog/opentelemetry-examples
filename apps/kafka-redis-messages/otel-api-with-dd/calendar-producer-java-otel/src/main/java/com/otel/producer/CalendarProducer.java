package com.otel.producer;

import com.otel.controller.CalendarController;
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.context.Scope;
import java.util.Random;
import java.util.concurrent.CountDownLatch;
import org.apache.kafka.clients.producer.Producer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * OTel API with DD backend variant: CalendarProducer uses the OTel API for manual
 * span creation, but the dd-java-agent provides the OTel API implementation.
 *
 * DD-specific: The dd-java-agent implements GlobalOpenTelemetry when
 * DD_TRACE_OTEL_ENABLED=true is set. Spans created via the OTel API are
 * converted to DD spans and sent to the DD Agent.
 *
 * DD_TRACE_PROPAGATION_STYLE_INJECT=tracecontext ensures W3C Trace Context
 * headers are injected into Kafka messages for cross-service propagation.
 *
 * The KafkaProducer is wrapped with KafkaTelemetry in KafkaConfig for
 * automatic context injection.
 */
@Component
public class CalendarProducer {
  private final Logger log = LoggerFactory.getLogger(CalendarController.class);
  private final Random random = new Random();
  private final Producer<String, String> producer;

  @Value("${kafka.topic:calendar}")
  private String topic;

  @Autowired
  public CalendarProducer(Producer<String, String> producer) {
    this.producer = producer;
  }

  public void write(String uuid) {
    // DD-specific: GlobalOpenTelemetry.getTracer() returns a DD-backed tracer
    // when DD_TRACE_OTEL_ENABLED=true.
    Span span = GlobalOpenTelemetry.getTracer("kafka-demo")
            .spanBuilder("write")
            .startSpan();
    try (Scope scope = span.makeCurrent()) {
      span.addEvent(uuid);
      span.setAttribute("example", "1");
      log.info("sending message:{}", uuid);
      ProducerRecord<String, String> record =
          new ProducerRecord<>(topic, String.valueOf(this.random.nextInt()), uuid);

      // Wait for send completion to capture errors in the span.
      CountDownLatch latch = new CountDownLatch(1);
      producer.send(record, (metadata, exception) -> {
        if (exception != null) {
          span.setStatus(StatusCode.ERROR, exception.getMessage());
          span.recordException(exception);
          log.error("Failed to send Kafka message", exception);
        }
        latch.countDown();
      });
      latch.await();
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      span.setStatus(StatusCode.ERROR, "Interrupted while sending");
      log.error("Interrupted while waiting for Kafka send", e);
    } finally {
      span.end();
    }
  }
}
