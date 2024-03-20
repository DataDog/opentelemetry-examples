/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.
*/
package com.otel.controller;

import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.metrics.Meter;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.SpanKind;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Scope;
import java.time.LocalDate;
import java.time.Month;
import java.time.format.DateTimeFormatter;
import java.util.Map;
import java.util.Random;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CalendarController {
  private final Logger log = LoggerFactory.getLogger(CalendarController.class);
  private final Tracer tracer;

  @Autowired
  CalendarController(OpenTelemetry openTelemetry) {
    tracer = openTelemetry.getTracer(CalendarController.class.getName());
    Meter meter = openTelemetry.getMeter(CalendarController.class.getName());
  }


  @GetMapping("/calendar")
  public Map<String, String> getDate(@RequestHeader MultiValueMap<String, String> headers) {

    String output = getDate();
    // the correct JSON output should put this in quotes. Spring does not, so let's put quotes here
    // by hand.
    return Map.of("date", output);
  }
  private String getDate() {
    Span span = tracer.spanBuilder("getDate").setAttribute("peer.service", "random-date-service").setSpanKind(SpanKind.CLIENT).startSpan();
    try (Scope scope = span.makeCurrent()) {
      // get back a random date in the year 2022
      int val = new Random().nextInt(365);
      LocalDate start = LocalDate.of(2022, Month.JANUARY, 1).plusDays(val);
      String output = start.format(DateTimeFormatter.ISO_LOCAL_DATE);
      span.setAttribute("date", output);
      log.info("generated date: {}" , output);
      return output;
    } finally {
      span.end();
    }
  }
}
