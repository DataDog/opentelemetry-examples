package com.astronomystore.ad;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import dev.openfeature.contrib.hooks.otel.TracesHook;
import dev.openfeature.sdk.Client;
import dev.openfeature.sdk.OpenFeatureAPI;
import dev.openfeature.sdk.ProviderEvent;

@Configuration
public class FeatureFlagsConfig {
    private static final Logger LOGGER = LoggerFactory.getLogger(FeatureFlagsConfig.class);

    private static final String AD_SERVICE_NAME = "ad";

    // The application code only uses the vendor-neutral OpenFeature API. Until a provider is registered, OpenFeature
    // uses a no-op provider and evaluations return their default values.
    @Bean
    public Client featureFlagsClient() {
        OpenFeatureAPI api = OpenFeatureAPI.getInstance();
        // For Jaeger, and any observability backend not integrated with OpenFeature: records each flag evaluation as a
        // `feature_flag` event on the current span, with the `feature_flag.key`, `feature_flag.variant`, and
        // `feature_flag.provider_name` OTel semantic conventions attributes.
        api.addHooks(new TracesHook());
        Client client = api.getClient(AD_SERVICE_NAME);
        client.on(ProviderEvent.PROVIDER_READY, event -> LOGGER.info(
                "Feature flags provider ready: provider={} domain={} message={}",
                event.getProviderName(), event.getDomain(), event.getMessage()));
        client.on(ProviderEvent.PROVIDER_CONFIGURATION_CHANGED, event -> LOGGER.info(
                "Feature flags configuration changed: provider={} domain={} flags_changed={} message={}",
                event.getProviderName(), event.getDomain(), event.getFlagsChanged(), event.getMessage()));
        client.on(ProviderEvent.PROVIDER_STALE, event -> LOGGER.warn(
                "Feature flags provider stale, using cached flag values: provider={} domain={} message={}",
                event.getProviderName(), event.getDomain(), event.getMessage()));
        client.on(ProviderEvent.PROVIDER_ERROR, event -> LOGGER.error(
                "Feature flags provider error: provider={} domain={} error_code={} message={}",
                event.getProviderName(), event.getDomain(), event.getErrorCode(), event.getMessage()));
        return client;
    }
}
