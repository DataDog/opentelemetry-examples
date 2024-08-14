/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.

This product includes software developed at Datadog (https://www.datadoghq.com/)
Copyright 2024 Datadog, Inc.
*/
package com.otel.controller;

import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

import com.otel.service.CalendarService;

import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.metrics.DoubleHistogram;
import io.opentelemetry.api.metrics.LongCounter;
import io.opentelemetry.api.metrics.Meter;
import io.opentelemetry.api.metrics.ObservableDoubleGauge;

@RestController
public class CalendarController {
    private final Logger log = LoggerFactory.getLogger(CalendarController.class);
    private final LongCounter hitsCounter;
    private final DoubleHistogram latency;
    private final ObservableDoubleGauge activeUsersGauge;
    private final AtomicLong activeUsersCounter;
    private final CalendarService calendarService;

    @Autowired
    CalendarController(OpenTelemetry openTelemetry, String serviceName, CalendarService service) {
        log.info("Starting CalendarController for {}", serviceName);
        
        Meter meter = openTelemetry.getMeter(CalendarController.class.getName());
        hitsCounter = meter.counterBuilder(serviceName + ".api.hits").build();
        latency = meter.histogramBuilder(serviceName + ".task.duration").build();
        
        activeUsersCounter = new AtomicLong();
        activeUsersGauge = meter.gaugeBuilder(serviceName + ".active.users.gauge").buildWithCallback(measurement -> measurement.record(activeUsersCounter.get()));
        
        calendarService = service;
    }

    @GetMapping("/calendar")
    public Map<String, String> getDate(@RequestHeader MultiValueMap<String, String> headers) {
        long startTime = System.currentTimeMillis();
        activeUsersCounter.incrementAndGet();
        
        try {
            hitsCounter.add(1);
            String output = calendarService.getDate();
            return Map.of("date", output);
        } finally {
            long endTime = System.currentTimeMillis();
            latency.record(endTime - startTime);
            activeUsersCounter.decrementAndGet();
        }
    }
}
