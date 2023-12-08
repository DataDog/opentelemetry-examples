package com.otel.producer;

import com.otel.controller.WordsController;
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
public class WordsProducer {
  private final Logger log = LoggerFactory.getLogger(WordsController.class);

  private final Producer<String, String> producer;

  @Value("${kafka.topic:words}")
  private String topic;

  @Autowired
  public WordsProducer(Producer<String, String> producer) {
    this.producer = producer;
  }

  public void write(String word) {
    Span span = GlobalOpenTelemetry.getTracer("kafka-demo").spanBuilder("write").startSpan();
    try (Scope scope = Context.current().with(span).makeCurrent()) {
      span.setAttribute("word",word);
      Random random = new Random();
      log.info("sending message:{}", word);
      ProducerRecord<String, String> record =
          new ProducerRecord<>(topic, String.valueOf(random.nextInt()), word);

      producer.send(record);
      span.end();
    }
  }
}
