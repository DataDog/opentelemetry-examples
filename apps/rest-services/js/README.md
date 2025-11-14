# js

This application demonstrates a Node.js API that can be auto instrumented with [OTel Node.js](https://opentelemetry.io/docs/instrumentation/js/getting-started/nodejs/).

It requires a Postgres instance to be running; see connection settings in `database.js`.

## API
```
POST /users 
Adds a new user. Fields: name, email
```

```
GET /users 
Gets all users
```

```
GET /users/:name
Gets user by name
```

```
PUT /users/:id
Updates user by ID (idempotent)
```

## Setup
Run the following to install dependencies:
```
npm install
```

Run the following to run the app with OTel auto instrumentation and export the traces to the console:
```
export OTEL_TRACES_EXPORTER=console
export NODE_OPTIONS="--require @opentelemetry/auto-instrumentations-node/register"
node app.js
```

To export the traces to an OTLP (HTTP/JSON) endpoint:
```
export OTEL_TRACES_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="http://<your endpoint>/v1/traces"
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="<your authentication headers>"
export NODE_OPTIONS="--require @opentelemetry/auto-instrumentations-node/register"
node app.js
```
