package com.astronomystore.ad;

import java.util.ArrayList;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.server.ResponseStatusException;

import dev.openfeature.sdk.Client;
import dev.openfeature.sdk.MutableContext;

import io.opentelemetry.api.common.AttributeKey;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.semconv.incubating.UserIncubatingAttributes;

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
            "SELECT id, name FROM catalog.products WHERE ? = ANY(string_to_array(categories, ','))";

    // Boolean feature flag, modeled on the OpenTelemetry Demo's adFailure flag: when it evaluates to true for a user,
    // their ad requests fail. Roll it out to some users to demo how an observability backend correlates an error rate
    // increase with a flag rollout.
    private static final String AD_FAILURE_FLAG = "ad-failure";

    private final JdbcTemplate jdbcTemplate;
    private final Client featureFlags;

    public AdController(JdbcTemplate jdbcTemplate, Client featureFlags) {
        this.jdbcTemplate = jdbcTemplate;
        this.featureFlags = featureFlags;
    }

    @GetMapping("/get-ads")
    public List<Ad> getAds(@RequestParam(value = "context_keys", required = false) String[] contextKeys,
            @RequestParam(value = "user_id", required = false) String userId,
            @RequestParam(value = "user_email", required = false) String userEmail) {
        String[] keys = contextKeys != null ? contextKeys : new String[0];
        // Logging the user's email is a bad practice, done on purpose to demo Datadog Sensitive Data Scanner.
        LOGGER.info("getAds invoked with context_keys={} user.id={} user.email={}", List.of(keys), userId, userEmail);

        // The targeting key keeps a user in the same variant across requests, for consistent percentage rollouts.
        // Under the load generator, userId is one of the frontend's 4 demo users, `user-1001` to `user-1004`, picked
        // at random per request.
        MutableContext flagContext = new MutableContext(userId != null ? userId : "anonymous");
        boolean authenticated = userId != null;
        flagContext.add("authenticated", authenticated);
        if (authenticated) {
            flagContext.add("user_id", userId);
        }
        if (featureFlags.getBooleanValue(AD_FAILURE_FLAG, false, flagContext)) {
            LOGGER.error("Ads unavailable for user.id={}: feature flag {} enabled", userId, AD_FAILURE_FLAG);
            throw new ResponseStatusException(HttpStatus.SERVICE_UNAVAILABLE,
                    "Ads unavailable, feature flag " + AD_FAILURE_FLAG + " enabled");
        }

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
        span.setAttribute(AttributeKey.stringArrayKey("store.ad.context_keys"), List.of(keys));
        span.setAttribute(AttributeKey.longKey("store.ad.count"), (long) ads.size());
        if (userId != null) {
            span.setAttribute(UserIncubatingAttributes.USER_ID, userId);
        }
        if (userEmail != null) {
            span.setAttribute(UserIncubatingAttributes.USER_EMAIL, userEmail);
        }

        return ads;
    }

    @GetMapping("/health")
    public void health() {
    }
}
