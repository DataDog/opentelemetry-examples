package com.astronomystore.checkout;

import com.fasterxml.jackson.annotation.JsonProperty;

public record PlaceOrderRequest(
        @JsonProperty("user_id") String userId,
        @JsonProperty("user_currency") String userCurrency,
        Address address,
        String email,
        @JsonProperty("credit_card") CreditCardInfo creditCard) {

    public record Address(
            @JsonProperty("street_address") String streetAddress,
            String city,
            String state,
            String country,
            @JsonProperty("zip_code") String zipCode) {
    }

    public record CreditCardInfo(
            @JsonProperty("credit_card_number") String creditCardNumber,
            @JsonProperty("credit_card_cvv") int creditCardCvv,
            @JsonProperty("credit_card_expiration_year") int creditCardExpirationYear,
            @JsonProperty("credit_card_expiration_month") int creditCardExpirationMonth) {
    }
}
