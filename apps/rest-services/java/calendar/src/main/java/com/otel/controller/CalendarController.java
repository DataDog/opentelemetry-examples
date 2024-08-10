package com.otel.controller;

import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.metrics.DoubleHistogram;
import io.opentelemetry.api.metrics.LongCounter;
import io.opentelemetry.api.metrics.Meter;
import io.opentelemetry.api.metrics.ObservableDoubleGauge;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.SpanKind;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Scope;
import java.time.LocalDate;
import java.time.Month;
import java.time.format.DateTimeFormatter;
import java.util.Map;
import java.util.Random;
import java.util.concurrent.atomic.AtomicLong;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;
import com.otel.main.SpringApp.CalendarMBean;

@RestController
public class CalendarController {

    private final Logger log = LoggerFactory.getLogger(CalendarController.class);
    private final Tracer tracer;
    private final LongCounter hitsCounter;
    private final DoubleHistogram latency;
    private final ObservableDoubleGauge activeUsersGauge;
    private final AtomicLong activeUsersCounter;
    private final Random random = new Random();
    private final CalendarMBean calendarMBean;

    @Autowired
    CalendarController(OpenTelemetry openTelemetry, String serviceName, CalendarMBean calendarMBean) {
        this.calendarMBean = calendarMBean;
        log.info("Starting CalendarController for {}", serviceName);
        tracer = openTelemetry.getTracer(CalendarController.class.getName());
        Meter meter = openTelemetry.meterBuilder(CalendarController.class.getName()).build();
        hitsCounter = meter.counterBuilder(serviceName + ".api.hits").build();
        latency = meter.histogramBuilder(serviceName + ".task.duration").build();
        activeUsersCounter = new AtomicLong();
        activeUsersGauge = meter.gaugeBuilder(serviceName + ".active.users.guage").buildWithCallback(measurement -> measurement.record(activeUsersCounter.get()));
    }

    @GetMapping("/calendar")
    public Map<String, String> getDate(@RequestHeader MultiValueMap<String, String> headers) {
        long startTime = System.currentTimeMillis();
        activeUsersCounter.incrementAndGet();
        try {
            hitsCounter.add(1);
            calendarMBean.incrementHitsCount();
            long endTime = System.currentTimeMillis();
            calendarMBean.addRequestLatency(endTime - startTime);
            log.info("generated JMX hit count: {}, latency: {}", calendarMBean.getHitsCount(), calendarMBean.getRequestLatency());

            String output = getDate();
            return Map.of("date", output);
        } finally {
            activeUsersCounter.decrementAndGet();
        }
    }

    private String getDate() {
        Span span = tracer.spanBuilder("getDate").setAttribute("peer.service", "random-date-service").setSpanKind(SpanKind.CLIENT).startSpan();
        try (Scope scope = span.makeCurrent()) {
            int val = new Random().nextInt(365);
            LocalDate start = LocalDate.of(2022, Month.JANUARY, 1).plusDays(val);
            String output = start.format(DateTimeFormatter.ISO_LOCAL_DATE);
            span.setAttribute("date", output);
            Thread.sleep(random.nextLong(1, 950));
            log.info("generated date: {}", output);
            return output;
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        } finally {
            span.end();
        }
    }
}
