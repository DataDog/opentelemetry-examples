package com.astronomystore.ad;

import java.util.ArrayList;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.jdbc.core.JdbcTemplate;
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

    // The context keys recognized here (telescopes, binoculars, books, planetariums) are matched against the
    // comma-separated "categories" column of astronomy-db's catalog.products table.
    private static final String ADS_BY_CATEGORY_SQL =
            "SELECT id, name FROM catalog.products WHERE string_to_array(categories, ',') @> ARRAY[?]";

    private final JdbcTemplate jdbcTemplate;

    public AdController(JdbcTemplate jdbcTemplate) {
        this.jdbcTemplate = jdbcTemplate;
    }

    @GetMapping("/get-ads")
    public List<Ad> getAds(@RequestParam(value = "context_keys", required = false) String[] contextKeys) {
        String[] keys = contextKeys != null ? contextKeys : new String[0];
        LOGGER.info("getAds invoked with context_keys={}", List.of(keys));

        List<Ad> ads = new ArrayList<>();
        for (String key : keys) {
            List<Ad> matched = jdbcTemplate.query(ADS_BY_CATEGORY_SQL,
                    (rs, rowNum) -> new Ad("/product/" + rs.getString("id"), "Check out our " + rs.getString("name") + "!"),
                    key);
            ads.addAll(matched);
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
