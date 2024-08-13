/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.

This product includes software developed at Datadog (https://www.datadoghq.com/)
Copyright 2024 Datadog, Inc.
*/
package com.otel.service;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.Random;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.SpanKind;
import io.opentelemetry.instrumentation.annotations.WithSpan;

@Component
public class CalendarService {
    private final Logger log = LoggerFactory.getLogger(CalendarService.class);
    private final Random random = new Random();

    @WithSpan(kind = SpanKind.CLIENT)
    public String getDate() {
        Span span = Span.current();
        span.setAttribute("peer.service", "random-date-service");
        
        // generate a random day within current year
        int day = new Random().nextInt(365);
        LocalDate date = LocalDate.now().withDayOfYear(1 + day);
        String output = date.format(DateTimeFormatter.ISO_LOCAL_DATE);

        span.setAttribute("date", output);

        try {
            // add random sleep
            Thread.sleep(random.nextLong(1, 950));
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        }
 
        log.info("generated date: {}", output);
        return output;
    }
}
