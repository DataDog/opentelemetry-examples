# Load Generator

This k6 Deployment continuously runs five virtual users. Each k6 run lasts 24
hours, after which the Deployment starts a replacement Pod so load generation
continues indefinitely. Each iteration calls:

- `GET /api/products` on `frontend`;
- `GET /api/ads` on `frontend`;
- `POST /api/checkout` on `frontend`, which proxies to `POST /checkout/place-order` on `checkout`; and
- `POST /fraud-detection/check-order` on `fraud-detection`.

Deploy it after the application services:

```sh
kubectl apply -f kubernetes.yaml
```

Set `FRONTEND_URL` or `FRAUD_DETECTION_URL` in the Deployment to target different service endpoints.
