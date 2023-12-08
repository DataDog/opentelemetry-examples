/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.
*/
package com.otel.controller;

import com.otel.producer.WordsProducer;
import java.io.UnsupportedEncodingException;
import java.net.URLDecoder;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.util.MultiValueMap;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class WordsController {
  private final Logger log = LoggerFactory.getLogger(WordsController.class);

  private final WordsProducer wordsProducer;

  @Autowired
  public WordsController( WordsProducer wordsProducer) {
    this.wordsProducer = wordsProducer;
  }

  @PostMapping("/words")
  public Map<String, String> words(@RequestBody String requestBody, @RequestHeader MultiValueMap<String, String> headers) throws UnsupportedEncodingException {
    String uuid = UUID.randomUUID().toString();
   String  decodedBody = URLDecoder.decode(requestBody, "UTF-8");

    log.info("request uuid:{}, body:{}", uuid, decodedBody);

    String[]words = decodedBody.split(",");
    for (String word :words ) {
      wordsProducer.write(word);
    }
    return new HashMap<>();
  }
}
