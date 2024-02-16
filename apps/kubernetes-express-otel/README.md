## Kubernetes, Express, and OpenTelemetry Example

This sample application demonstrates a sample application started with the [Express tutorial](https://expressjs.com/en/starter/installing.html) with the instrumentation instructions from the [OpenTelemetry Node.js tutorial](https://opentelemetry.io/docs/languages/js/getting-started/nodejs/) to demonstrate how traces can be submitted in Kubernetes via OTLP.


### Example Trace
[!sample-express-otel-trace](sample-express-otel-trace.jpeg)


### Prerequistes
1. Datadog API Key and Datadog App Key
2. Kubernetes

### Spin up a Datadog Agent Pod

#### Option 1: Datadog Operator

The latest instructions can be found under [Install the Datadog Agent on Kubernetes](https://docs.datadoghq.com/containers/kubernetes/installation/?tab=operator#deploy-an-agent-with-the-operator).

**Note:** Please replace the **site** in `/datadog-operator/datadog-agent.yaml` with the correct site as needed.

Commands to spin up the Datadog Agent pods

```
helm repo add datadog https://helm.datadoghq.com
helm install my-datadog-operator datadog/datadog-operator

kubectl create secret generic datadog-secret --from-literal api-key=<DATADOG_API_KEY> --from-literal app-key=<DATADOG_APP_KEY>

kubectl apply -f ./datadog-operator/datadog-agent.yaml

```

Wait a few minutes.

Then, run `kubectl get pods` 

You should see the Datadog Agent pods like so:


```
NAME                                     READY   STATUS    RESTARTS   AGE
datadog-agent-****                      3/3     Running   0          66s
datadog-cluster-agent-**********-*****   1/1     Running   0          69s
my-datadog-operator-**********-*****     1/1     Running   0          2m30s
```

### Spin up a NodeJS application

This sample app builds a local copy of the app image so that it doesn't need to be pushed to another repository.

```
docker build -t otel-js-app:1.0 ./express-otel-app/
kubectl apply -f ./express-otel-app/app.yaml

```

When you run `kubectl get pods`, you should see the app pod:

```
NAME                                     READY   STATUS    RESTARTS   AGE
app-**********-*****                     1/1     Running   0          5m17s
datadog-agent-*****                      3/3     Running   0          26m
datadog-cluster-agent-**********-*****   1/1     Running   0          26m
my-datadog-operator-**********-*****     1/1     Running   0          27m
```

### Example Endpoints

This can be done by execing into the app pod and sending manual requests:

1. `curl localhost:3000/`
2. `curl localhost:3000/error`

### Spin down everything

**Datadog Operator**
```
kubectl delete datadogagent datadog
helm delete my-datadog-operator
```

**Application pod**
```
kubectl delete -f ./express-otel-app/app.yaml
```

### Resources

1. https://expressjs.com/en/starter/installing.html
1. https://opentelemetry.io/docs/languages/js/getting-started/nodejs/
1. https://opentelemetry.io/docs/languages/js/exporters/
1. https://docs.datadoghq.com/containers/kubernetes/installation/?tab=operator#deploy-an-agent-with-the-operator
1. https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#general-sdk-configuration