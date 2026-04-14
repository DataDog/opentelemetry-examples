# Collector configuration generator

An internal tool using Go templates to generate the configurations in `configurations/opentelemetry-collector`.

To regenerate the configurations, `cp` into the `generator` subdirectory and run:
```sh
go run . templates ..
```
