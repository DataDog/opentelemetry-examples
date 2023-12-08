package com.otel.producer;

import com.otel.controller.CalendarController;
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.context.Context;
import io.opentelemetry.context.Scope;
import java.util.Random;
import org.apache.kafka.clients.producer.Producer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class CalendarProducer {
  private final Logger log = LoggerFactory.getLogger(CalendarController.class);

  private final Producer<String, String> producer;

  @Value("${kafka.topic:calendar}")
  private String topic;

  @Autowired
  public CalendarProducer(Producer<String, String> producer) {
    this.producer = producer;
  }

  public void write(String uuid) {
    Span span = GlobalOpenTelemetry.getTracer("kafka-demo").spanBuilder("write").startSpan();
    try (Scope scope = Context.current().with(span).makeCurrent()) {
      span.addEvent(uuid);
      Random random = new Random();
      log.info("sending message:{}", uuid);
      ProducerRecord<String, String> record =
          new ProducerRecord<>(topic, String.valueOf(random.nextInt()), uuid);

      producer.send(record);
      span.end();
    }
  }
}
