package com.otel.producer;

import com.otel.controller.CalendarController;
import io.opentracing.Span;
import io.opentracing.util.GlobalTracer;
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
 * DD-specific: CalendarProducer uses the OpenTracing API for manual span creation.
 * The dd-java-agent auto-instruments the KafkaProducer, injecting DD trace context
 * into Kafka message headers for distributed trace propagation to the consumer.
 *
 * The dd-java-agent automatically maps Kafka messaging attributes:
 * - messaging.system -> kafka
 * - messaging.destination -> topic name
 * - messaging.kafka.partition -> partition number
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
    // DD-specific: GlobalTracer is provided by the dd-java-agent at runtime.
    // The span is automatically linked to the active HTTP request span.
    Span span = GlobalTracer.get().buildSpan("kafka-producer").withTag("uuid", uuid).start();
    try {
      log.info("sending message:{}", uuid);
      ProducerRecord<String, String> record =
          new ProducerRecord<>(topic, String.valueOf(random.nextInt()), uuid);

      // Use a latch to wait for the send to complete, ensuring the span
      // captures any errors from the Kafka broker.
      CountDownLatch latch = new CountDownLatch(1);
      producer.send(record, (metadata, exception) -> {
        if (exception != null) {
          log.error("Failed to send Kafka message", exception);
          span.setTag("error", true);
          span.log("Kafka send failed: " + exception.getMessage());
        }
        latch.countDown();
      });
      latch.await();
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      log.error("Interrupted while waiting for Kafka send", e);
    } finally {
      span.finish();
    }
  }
}
