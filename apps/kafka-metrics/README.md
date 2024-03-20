# Kafka Metrics

This project consists of a Java Producer and Consumer, a kafka broker, a zookeeper instance and the OpenTelemetry Collector and JMX Metrics Gatherer.

The Producer produces messages to topic orders, which the consumer consumes. The producer outputs the order ID it produced, and the consumer outputs the order ID it consumed, e.g.
```
producer                   | log4j2: 14:07:45.306 [main] INFO  Producer - Message (order # 13) sent successfully!
consumer                   | log4j2: 14:07:45.334 [main] INFO  Consumer - Consumed Message. Received Order # 13
producer                   | log4j2: 14:07:49.312 [main] INFO  Producer - Message (order # 14) sent successfully!
consumer                   | log4j2: 14:07:49.342 [main] INFO  Consumer - Consumed Message. Received Order # 14
producer                   | log4j2: 14:07:53.320 [main] INFO  Producer - Message (order # 15) sent successfully!
consumer                   | log4j2: 14:07:53.349 [main] INFO  Consumer - Consumed Message. Received Order # 15
producer                   | log4j2: 14:07:57.324 [main] INFO  Producer - Message (order # 16) sent successfully!
consumer                   | log4j2: 14:07:57.356 [main] INFO  Consumer - Consumed Message. Received Order # 16
producer                   | log4j2: 14:08:01.327 [main] INFO  Producer - Message (order # 17) sent successfully!
consumer                   | log4j2: 14:08:01.368 [main] INFO  Consumer - Consumed Message. Received Order # 17
producer                   | log4j2: 14:08:05.332 [main] INFO  Producer - Message (order # 18) sent successfully!
consumer                   | log4j2: 14:08:05.336 [main] INFO  Consumer - Consumed Message. Received Order # 18
producer                   | log4j2: 14:08:09.336 [main] INFO  Producer - Message (order # 19) sent successfully!
consumer                   | log4j2: 14:08:09.361 [main] INFO  Consumer - Consumed Message. Received Order # 19
producer                   | log4j2: 14:08:13.341 [main] INFO  Producer - Message (order # 20) sent successfully!
consumer                   | log4j2: 14:08:13.360 [main] INFO  Consumer - Consumed Message. Received Order # 20
```

This examples collects kafka metrics using the kafka metrics receiver and the JMX Receiver or JMX Metrics Gatherer to collect the JMX based kafka metrics. These metrics give access to the "Kafka, Zookeeper and Kafka Consumer Overview" OOTB Dashboard.

- `docker-compose.jmxreceiver.yaml` showcases how to collect the JMX Based kafka metrics using the JMX Receiver.
{{< img src="/assets/jmxreceiver.png" alt="OpenTelemetry Kafka metrics via jmx receiver" style="width:70%;" >}}

- `docker-compose.jmxmetricsgatherer.yaml` showcases how to collect the JMX Based kafka metrics using the JMX Metrics Gatherer.
{{< img src="/assets/jmxmetricsgatherer.png" alt="OpenTelemetry Kafka metrics via jmx metrics gatherer" style="width:70%;" >}}

Both have a collector using the kafka metrics receiver.

In addition, the producer and consumer send logs via the OTLP exporter to the Collector. These logs are tagged by `source:kafka` by an attributes processor, and will show up in the "Kafka, Zookeeper and Kafka Consumer Overview" OOTB Dashboard.


*Note:* Metrics `kafka.request.fetch_follower.time.avg`, `kafka.request.fetch_consumer.time.avg`, and `kafka.request.produce.time.avg` will be missing until v1.33.0 of [opentelemetry-jmx-metrics](https://github.com/open-telemetry/opentelemetry-java-contrib/releases) is released.

## Docker Compose
Retrieve your API_KEY from datadoghq, and expose your key on the shell:
```
export DD_API_KEY=xx
```

*JMX RECEIVER:*

Bring up the client, server & collector:
```
docker-compose -f docker-compose.jmxreceiver.yaml build
docker-compose -f docker-compose.jmxreceiver.yaml up
```

Spin down the client, server & collector:
```
docker -f docker-compose.jmxreceiver.yaml compose down || Ctrl+C
```

*JMX METRICS GATHERER:*

Bring up the client, server, collector and JMX Metrics Gatherer:
```
docker-compose -f docker-compose.jmxmetricsgatherer.yaml build
docker-compose -f docker-compose.jmxmetricsgatherer.yaml up
```

Spin down the client, server, collector and JMX Metrics Gatherer:
```
docker -f docker-compose.jmxmetricsgatherer.yaml compose down || Ctrl+C
```