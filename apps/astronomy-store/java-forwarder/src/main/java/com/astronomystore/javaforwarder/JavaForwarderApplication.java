package com.astronomystore.javaforwarder;

import java.time.Duration;
import java.util.List;
import java.util.Properties;
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

public class JavaForwarderApplication {
    private static final Logger logger = LoggerFactory.getLogger(JavaForwarderApplication.class);
    private static final String CONSUMER_GROUP = "java-forwarder";

    public static void main(String[] args) throws InterruptedException {
        String broker = env("KAFKA_ADDR", "kafka:9092");
        String inputTopic = env("KAFKA_TOPIC", "python-forwarder-orders");
        String outputTopic = env("KAFKA_OUTPUT_TOPIC", "java-forwarder-orders");

        Properties consumerProps = new Properties();
        consumerProps.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, broker);
        consumerProps.put(ConsumerConfig.GROUP_ID_CONFIG, CONSUMER_GROUP);
        consumerProps.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
        consumerProps.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG, StringDeserializer.class.getName());
        consumerProps.put(ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, StringDeserializer.class.getName());

        Properties producerProps = new Properties();
        producerProps.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, broker);
        producerProps.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
        producerProps.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());

        try (KafkaConsumer<String, String> consumer = new KafkaConsumer<>(consumerProps);
             KafkaProducer<String, String> producer = new KafkaProducer<>(producerProps)) {
            consumer.subscribe(List.of(inputTopic));
            logger.info("java-forwarder started: {} -> {}", inputTopic, outputTopic);
            while (true) {
                for (ConsumerRecord<String, String> record : consumer.poll(Duration.ofSeconds(1))) {
                    logger.info("forward_order key={}", record.key());
                    Thread.sleep(processingDelayMillis());
                    producer.send(new ProducerRecord<>(outputTopic, record.key(), record.value()));
                }
            }
        }
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
