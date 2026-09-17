# Load Generator

This k6 Deployment continuously runs five virtual users. Each k6 run lasts 24
hours, after which the Deployment starts a replacement Pod so load generation
continues indefinitely. Each iteration calls:

- `GET /api/products` on `frontend`;
- `GET /api/ads` on `frontend`; and
- `POST /api/checkout` on `frontend`, which proxies to `POST /checkout/place-order` on `checkout`.

`checkout` publishes each placed order to the `orders` Kafka topic, which `fraud-detection` consumes
asynchronously — it is not called directly by the load generator.

Deploy it after the application services:

```sh
kubectl apply -f kubernetes.yaml
```

Set `FRONTEND_URL` in the Deployment to target a different frontend endpoint.
