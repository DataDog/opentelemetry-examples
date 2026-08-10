# Load Generator

This k6 Deployment runs five virtual users for ten minutes. Each iteration calls:

- `GET /api/products` on `frontend`;
- `POST /checkout/place-order` on `checkout`; and
- `POST /fraud-detection/check-order` on `fraud-detection`.

Deploy it after the application services:

```sh
kubectl apply -f kubernetes.yaml
```

Set `FRONTEND_URL`, `CHECKOUT_URL`, or `FRAUD_DETECTION_URL` in the Deployment to target different service endpoints.
