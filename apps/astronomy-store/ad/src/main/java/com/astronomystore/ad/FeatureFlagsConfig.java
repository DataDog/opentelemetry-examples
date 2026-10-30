package com.astronomystore.ad;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import datadog.trace.api.openfeature.Provider;
import dev.openfeature.contrib.hooks.otel.TracesHook;
import dev.openfeature.sdk.Client;
import dev.openfeature.sdk.OpenFeatureAPI;
import dev.openfeature.sdk.exceptions.OpenFeatureError;

@Configuration
public class FeatureFlagsConfig {
    private static final Logger LOGGER = LoggerFactory.getLogger(FeatureFlagsConfig.class);

    private static final String AD_SERVICE_NAME = "ad";

    @Bean
    public Client featureFlagsClient() {
        OpenFeatureAPI api = OpenFeatureAPI.getInstance();
        api.addHooks(new TracesHook());

        try {
            api.setProviderAndWait(new Provider());
        } catch (OpenFeatureError e) {
            LOGGER.warn("Feature flags provider not ready, using the default flag values: {}", e.getMessage());
        }
        return api.getClient(AD_SERVICE_NAME);
    }
}
