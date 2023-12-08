# js
This application demonstrates a Node.js API that can be auto instrumented with [OTel Node.js](https://opentelemetry.io/docs/instrumentation/js/getting-started/nodejs/).

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

Run the following to run the app with OTel auto instrumentation:
```
export OTEL_TRACES_EXPORTER=console
export NODE_OPTIONS="--require @opentelemetry/auto-instrumentations-node/register"
node app.js

```
