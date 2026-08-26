package com.astronomystore.ad;

import com.fasterxml.jackson.annotation.JsonProperty;

public record Ad(@JsonProperty("redirect_url") String redirectUrl, String text) {
}
