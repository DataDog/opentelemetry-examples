/*
 * This Java source file was generated by the Gradle 'init' task.
 */
package producer;

import java.util.Properties;
import java.lang.Thread;
import java.lang.InterruptedException;

import org.apache.kafka.common.serialization.StringSerializer;
import org.apache.kafka.common.serialization.IntegerSerializer;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.Producer;

import io.opentelemetry.sdk.OpenTelemetrySdk;
import io.opentelemetry.sdk.autoconfigure.AutoConfiguredOpenTelemetrySdk;

import org.apache.logging.log4j.LogManager;


public class App {
    private static final org.apache.logging.log4j.Logger log4jLogger = LogManager.getLogger("Producer");

    public static void main(String[] args) {
        OpenTelemetrySdk sdk = AutoConfiguredOpenTelemetrySdk.initialize().getOpenTelemetrySdk();
        io.opentelemetry.instrumentation.log4j.appender.v2_17.OpenTelemetryAppender.install(sdk);

        String kafkaAddr = System.getenv("KAFKA_SERVICE_ADDR");
        if (kafkaAddr != null) {
            log4jLogger.info("Using Kafka Broker Address: " + kafkaAddr);
        } else {
            throw new RuntimeException("Environment variable KAFKA_SERVICE_ADDR is not set.");
        }

        Properties props = new Properties();
        props.put("bootstrap.servers", kafkaAddr);
        props.setProperty(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class.getName());
        props.setProperty(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, IntegerSerializer.class.getName());

        Producer<String, Integer> producer = new KafkaProducer<>(props);
        Integer orderNum = 0;

        while (true) {
            orderNum++;
            try {
                ProducerRecord<String, Integer> record = new ProducerRecord<>("orders", "order-number", orderNum);
                producer.send(record);
            } catch (Exception e) {
                log4jLogger.error("Unable to send record: ", e);
            } finally {
                log4jLogger.info("Message (order # " + orderNum + ") sent successfully!");
            }
            try {
                Thread.sleep(4000);
            } catch (InterruptedException e) {
                log4jLogger.error("Unable to sleep: ", e);
            }
        }
        // producer.close();
    }
}
