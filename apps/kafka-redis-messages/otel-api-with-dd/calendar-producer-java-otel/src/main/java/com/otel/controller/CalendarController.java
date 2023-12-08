/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.
*/
package com.otel.controller;

import com.otel.producer.CalendarProducer;
import java.util.Map;
import java.util.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;
import redis.clients.jedis.Jedis;

@RestController
public class CalendarController {
  private final Logger log = LoggerFactory.getLogger(CalendarController.class);

  private final CalendarProducer calendarProducer;
  private final Jedis jedis;

  @Autowired
  public CalendarController(Jedis jedis, CalendarProducer calendarProducer) {
    this.calendarProducer = calendarProducer;
    this.jedis = jedis;
  }

  @GetMapping("/calendar")
  public Map<String, String> getDate(@RequestHeader MultiValueMap<String, String> headers)
      throws InterruptedException {

    String uuid = UUID.randomUUID().toString();
    log.info("request uuid:{}", uuid);
    calendarProducer.write(uuid);
    int cnt = 0;
    var value = jedis.get(uuid);
    while (value == null && cnt < 30) {
      Thread.sleep(100);
      value = jedis.get(uuid);

      cnt++;
    }
    // the correct JSON output should put this in quotes. Spring does not, so let's put quotes here
    // by hand.
    if (value == null) {
      return Map.of("date", "null");
    }
    return Map.of("date", value.toString());
  }
}
