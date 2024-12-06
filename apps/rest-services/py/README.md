## Sample CRUD APPs

Sample python apps that provide CRUD api for Users table for both SQLite & Postgres.

Command to run the app with opentelemetry-instrument and export traces to console
```
 opentelemetry-instrument \
    --traces_exporter console \
    --metrics_exporter console \
    --service_name my-service \
    python app_sqlite.py

```
