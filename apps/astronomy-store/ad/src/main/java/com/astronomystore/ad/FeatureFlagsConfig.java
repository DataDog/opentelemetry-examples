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

    // The application code only uses the vendor-neutral OpenFeature API, this configuration plugs in the Datadog
    // OpenFeature provider. Swapping the provider (e.g. for flagd, as in the OpenTelemetry Demo) needs no other change.
    @Bean
    public Client featureFlagsClient() {
        OpenFeatureAPI api = OpenFeatureAPI.getInstance();
        // For Jaeger, and any observability backend not integrated with OpenFeature: records each flag evaluation as a
        // `feature_flag` event on the current span, with the `feature_flag.key`, `feature_flag.variant`, and
        // `feature_flag.provider_name` OTel semantic conventions attributes.
        api.addHooks(new TracesHook());

        try {
            // Set provider and wait for initial configuration (recommended). The injected dd-java-agent doesn't register
            // the Datadog provider automatically. See https://docs.datadoghq.com/feature_flags/server/java/ .
            api.setProviderAndWait(new Provider());
        } catch (OpenFeatureError e) {
            // Keep starting the application: evaluations return their default values until the provider gets its
            // configuration.
            LOGGER.warn("Feature flags provider not ready, using the default flag values: {}", e.getMessage());
        }
        return api.getClient(AD_SERVICE_NAME);
    }
}
