# Checkout

This Spring Boot service accepts a checkout request at `POST /checkout/place-order`. Its JSON request body mirrors the OpenTelemetry Demo [`PlaceOrderRequest`](https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto#L228-L235) structure:

```json
{
  "user_id": "user-123",
  "user_currency": "USD",
  "address": {
    "street_address": "123 Astronomy Way",
    "city": "Paris",
    "state": "Ile-de-France",
    "country": "FR",
    "zip_code": "75001"
  },
  "email": "user@example.com",
  "credit_card": {
    "credit_card_number": "4111111111111111",
    "credit_card_cvv": 123,
    "credit_card_expiration_year": 2030,
    "credit_card_expiration_month": 12
  }
}
```

The service returns `201 Created` with a JSON `order_id`. Build and deploy it with:

```sh
docker build -t astronomy-store/checkout:latest .
kubectl apply -f kubernetes.yaml
```

The Deployment uses the OpenTelemetry Operator's Java auto-instrumentation annotation. The application does not configure an OpenTelemetry SDK; the Operator instruments Spring MVC HTTP requests.
