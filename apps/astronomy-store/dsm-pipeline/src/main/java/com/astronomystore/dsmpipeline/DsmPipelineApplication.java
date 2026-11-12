package com.astronomystore.dsmpipeline;

import java.time.Duration;
import java.util.List;
import java.util.Properties;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;

import org.apache.kafka.clients.consumer.ConsumerConfig;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.apache.kafka.clients.consumer.KafkaConsumer;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.common.serialization.StringDeserializer;
import org.apache.kafka.common.serialization.StringSerializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DsmPipelineApplication {
    private static final Logger logger = LoggerFactory.getLogger(DsmPipelineApplication.class);

    public static void main(String[] args) throws InterruptedException {
        String role = env("ROLE", "");
        String broker = env("KAFKA_ADDR", "kafka:9092");
        switch (role) {
            case "producer" -> produce(broker, env("KAFKA_OUTPUT_TOPIC", "dsm-orders"));
            case "forwarder" -> forward(broker, env("KAFKA_TOPIC", "dsm-orders"), env("KAFKA_OUTPUT_TOPIC", "dsm-orders-forwarded"));
            case "consumer" -> consume(broker, env("KAFKA_TOPIC", "dsm-orders-forwarded"));
            default -> throw new IllegalArgumentException("ROLE must be one of: producer, forwarder, consumer (got '" + role + "')");
        }
    }

    private static void produce(String broker, String outputTopic) throws InterruptedException {
        try (KafkaProducer<String, String> producer = new KafkaProducer<>(producerProps(broker))) {
            logger.info("dsm-producer started: -> {}", outputTopic);
            while (true) {
                String key = UUID.randomUUID().toString();
                producer.send(new ProducerRecord<>(outputTopic, key, "{\"order_id\":\"" + key + "\"}"));
                logger.info("produce_order key={}", key);
                Thread.sleep(1000);
            }
        }
    }

    private static void forward(String broker, String inputTopic, String outputTopic) throws InterruptedException {
        try (KafkaConsumer<String, String> consumer = new KafkaConsumer<>(consumerProps(broker, "dsm-forwarder"));
             KafkaProducer<String, String> producer = new KafkaProducer<>(producerProps(broker))) {
            consumer.subscribe(List.of(inputTopic));
            logger.info("dsm-forwarder started: {} -> {}", inputTopic, outputTopic);
            while (true) {
                for (ConsumerRecord<String, String> record : consumer.poll(Duration.ofSeconds(1))) {
                    Thread.sleep(processingDelayMillis());
                    producer.send(new ProducerRecord<>(outputTopic, record.key(), record.value()));
                    logger.info("forward_order key={}", record.key());
                }
            }
        }
    }

    private static void consume(String broker, String inputTopic) {
        try (KafkaConsumer<String, String> consumer = new KafkaConsumer<>(consumerProps(broker, "dsm-consumer"))) {
            consumer.subscribe(List.of(inputTopic));
            logger.info("dsm-consumer started: {} ->", inputTopic);
            while (true) {
                for (ConsumerRecord<String, String> record : consumer.poll(Duration.ofSeconds(1))) {
                    logger.info("consume_order key={}", record.key());
                }
            }
        }
    }

    private static Properties consumerProps(String broker, String group) {
        Properties props = new Properties();
        props.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, broker);
        props.put(ConsumerConfig.GROUP_ID_CONFIG, group);
        props.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
        props.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG, StringDeserializer.class.getName());
        props.put(ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, StringDeserializer.class.getName());
        return props;
    }

    private static Properties producerProps(String broker) {
        Properties props = new Properties();
        props.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, broker);
        props.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
        props.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
        return props;
    }

    private static long processingDelayMillis() {
        double millis = 500 + 150 * ThreadLocalRandom.current().nextGaussian();
        return (long) Math.min(Math.max(millis, 0), 990);
    }

    private static String env(String key, String defaultValue) {
        String value = System.getenv(key);
        return value == null || value.isEmpty() ? defaultValue : value;
    }
}
