package com.astronomystore.ad;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import io.opentelemetry.api.common.AttributeKey;
import io.opentelemetry.api.trace.Span;

@RestController
@RequestMapping("/ad")
public class AdController {
    private static final Logger LOGGER = LoggerFactory.getLogger(AdController.class);

    private static final List<Ad> DEFAULT_ADS = List.of(
            new Ad("/product/telescope-explorer-150", "150mm reflector telescopes are 20% off this week."),
            new Ad("/product/star-map-poster", "Get a free star map poster with every purchase."));

    private static final Map<String, List<Ad>> ADS_BY_CONTEXT_KEY = Map.of(
            "telescopes", List.of(
                    new Ad("/product/telescope-explorer-150", "150mm reflector telescopes are 20% off this week.")),
            "binoculars", List.of(
                    new Ad("/product/binoculars-astro-10x50", "10x50 astronomy binoculars, in stock now.")),
            "books", List.of(
                    new Ad("/product/field-guide-to-the-night-sky", "The best-selling field guide, now in paperback.")),
            "planetariums", List.of(
                    new Ad("/product/home-planetarium-pro", "Bring the night sky indoors with our Planetarium Pro.")));

    @GetMapping("/get-ads")
    public List<Ad> getAds(@RequestParam(value = "context_keys", required = false) String[] contextKeys) {
        String[] keys = contextKeys != null ? contextKeys : new String[0];
        LOGGER.info("getAds invoked with context_keys={}", List.of(keys));

        List<Ad> ads = new ArrayList<>();
        for (String key : keys) {
            List<Ad> matched = ADS_BY_CONTEXT_KEY.get(key);
            if (matched != null) {
                ads.addAll(matched);
            }
        }
        if (ads.isEmpty()) {
            ads.addAll(DEFAULT_ADS);
        }

        Span span = Span.current();
        span.setAttribute(AttributeKey.stringArrayKey("app.ads.context_keys"), List.of(keys));
        span.setAttribute(AttributeKey.longKey("app.ads.count"), (long) ads.size());

        return ads;
    }

    @GetMapping("/health")
    public void health() {
    }
}
