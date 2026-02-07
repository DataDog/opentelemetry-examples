/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.
*/
package com.otel.controller;

import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestTemplate;

/**
 * Calendar controller that demonstrates W3C Trace Context propagation.
 *
 * <p>When instrumented with either the OpenTelemetry Java agent or the Datadog
 * Java agent, the {@link RestTemplate} automatically injects {@code traceparent}
 * and {@code tracestate} headers into outgoing HTTP requests. The downstream
 * Python service (instrumented with either OTel or ddtrace) extracts these
 * headers and continues the distributed trace.
 *
 * <p>No manual header propagation is required -- both agents handle injection
 * and extraction of W3C Trace Context headers transparently.
 */
@RestController
public class CalendarController {
  private static final Logger log = LoggerFactory.getLogger(CalendarController.class);
  private final RestTemplate restTemplate = new RestTemplate();

  @Value("${calendar.service.url}")
  private String calendarServiceUrl;

  @GetMapping("/calendar")
  public ResponseEntity<Map<String, String>> getDate() {
    log.info("Making call to downstream calendar service at {}", calendarServiceUrl);
    try {
      String date = restTemplate.getForObject(calendarServiceUrl + "/calendar", String.class);
      if (date == null) {
        log.warn("Received null response from downstream calendar service");
        return ResponseEntity.internalServerError()
            .body(Map.of("error", "no date received from downstream service"));
      }
      log.info("Date retrieved: {}", date);
      return ResponseEntity.ok(Map.of("date", date));
    } catch (RestClientException e) {
      log.error("Failed to call downstream calendar service", e);
      return ResponseEntity.internalServerError()
          .body(Map.of("error", "failed to reach downstream service"));
    }
  }

  @GetMapping("/health")
  public ResponseEntity<Map<String, String>> health() {
    return ResponseEntity.ok(Map.of("status", "healthy"));
  }
}
