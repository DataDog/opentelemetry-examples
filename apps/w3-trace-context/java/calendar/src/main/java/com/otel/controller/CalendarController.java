/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.
*/
package com.otel.controller;

import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;

@RestController
public class CalendarController {
  private final Logger log = LoggerFactory.getLogger(CalendarController.class);
  private final RestTemplate restTemplate = new RestTemplate();

  @Value("${calendar.service.url}")
  private String calendarServiceUrl;

  @GetMapping("/calendar")
  public Map<String, String> getDate(@RequestHeader MultiValueMap<String, String> headers) {

    log.info("making call to service url:{}", calendarServiceUrl);
    var date = restTemplate.getForObject(calendarServiceUrl + "/calendar", String.class);
    log.info("date retrieved:{}", date);
    return Map.of("date", date);
  }
}
