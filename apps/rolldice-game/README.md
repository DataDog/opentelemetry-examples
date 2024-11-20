# W3C Trace Context Example
This project consists of two Flask servers and one Express server instrumented with Opentelemetry on a standalone host. (This is not using k8s or containers)

## Start the Demo
### Option 1 (Standalone Host)
There are a few required steps to get this example working. 
1. Install and set up the OpenTelemetry Collector on the host of your choice. https://opentelemetry.io/docs/collector/installation/
2. Set up the Collector configuration. An example collector configuration can be found at [Config File](./config.yml).
    * The config.yaml is set up to send traces and metrics from OTLP Collector to the Datadog Exporter.
    * Update the DATADOG_API_KEY as well.
3. Run this collector  ~ `./otelcol-contrib --config=config.yaml`
4. Once the collector is up and running. Set up the 3 servers. (Can be found in each of the servers README)

### Option 2 (Docker)
1. Update DATADOG_API_KEY in [Config File](./config.yml).
2. Run docker-compose up

To trigger the service call

### Success
```bash
curl -X POST http://localhost:5002/play_game \
     -H "Content-Type: application/json" \
     -d '{"player": "John Doe"}'
```
### Error
``` bash
curl -X POST http://localhost:5002/play_game \
     -H "Content-Type: application/json" \
     -d '{}'
```
View traces & standalone host in app.datadoghq.com
