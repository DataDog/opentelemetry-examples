## Kubernetes, Express, and OpenTelemetry Example

This sample application demonstrates a sample application started with the [Express tutorial](https://expressjs.com/en/starter/installing.html) with the instrumentation instructions from the [OpenTelemetry Node.js tutorial](https://opentelemetry.io/docs/languages/js/getting-started/nodejs/) to demonstrate how traces can be submitted in Kubernetes via OTLP.


### Example Trace
[!sample-express-otel-trace](sample-express-otel-trace.jpeg)


### Prerequistes
1. Datadog API Key and Datadog App Key
2. Kubernetes

### Spin up a Datadog Agent Pod

Choose either **Option 1: Via Datadog Operator** OR **Option 2: Via Helm**. 

#### Option 1: Via Datadog Operator

The latest instructions can be found under [Install the Datadog Agent on Kubernetes (Operator)](https://docs.datadoghq.com/containers/kubernetes/installation/?tab=operator#deploy-an-agent-with-the-operator).

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

#### Option 2: Via Helm

The latest instructions can be found under [Install the Datadog Agent on Kubernetes (Helm)](https://docs.datadoghq.com/containers/kubernetes/installation/?tab=helm#deploy-an-agent-with-the-operator).

**Note:** Please replace the **site** in `/helm/datadog-values.yaml` with the correct site as needed.

```
helm install datadog-agent -f ./helm/datadog-values.yaml --set targetSystem=linux datadog/datadog
```

After waiting a few minutes, you can run `kubectl get pods`:

```
kubectl get pods
NAME                                           READY   STATUS     RESTARTS   AGE
datadog-agent-cluster-agent-**********-*****   1/1     Running   0          51s
datadog-agent-*****                           1/1     Running   0          51s
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

There are two main endpoints that can be used inside the pod:

1. `curl localhost:3000/`
2. `curl localhost:3000/error`

To exec into the app pod and send manual requests:

To do this, run the command below to get a list of pods, then find the app pod name, ie: `app-**********-*****`:

```
kubectl get pods
```

Then have the app pod send a request to one of the configured endpoints, ie:

```
kubectl exec -it <app pod name> -- curl localhost:3000/
```

If you check the account the API and APP Key was configured with, you should see now see traces.


### Spin down everything

**Datadog Operator**
```
kubectl delete datadogagent datadog
helm delete my-datadog-operator
```

**Helm**

```
helm uninstall datadog-agent
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