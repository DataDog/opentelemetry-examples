package com.astronomystore.checkout;

import com.fasterxml.jackson.annotation.JsonProperty;

public record PlaceOrderResponse(@JsonProperty("order_id") String orderId) {
}
