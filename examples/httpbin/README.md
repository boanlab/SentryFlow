# Single HTTP Requests

This document demonstrates how SentryFlow effectively captures API logs for simple HTTP requests, using Istio's `sleep` and `httpbin` examples for illustration.

It is essential to ensure that the `sleep` and `httpbin` deployments are correctly configured and that the default namespace has [Istio injection enabled](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection) for the setup to function properly.

## Step 1. Verify Services

To confirm that Istio is set up correctly, start by verifying if the `default` namespace has Istio injection enabled. This can be done using the following command:

```bash
kubectl describe namespace default

Name:         default
Labels:       istio-injection=enabled
```

If the namespace `default` has label `istio-injection=enabled`, this was set properly. Now, apply the `telemetry.yaml` in this directory by following command:

```bash
kubectl create -f telemetry.yaml

telemetry.telemetry.istio.io/sleep-logging created
```

Executing this command will configure `telemetry` for Istio, instructing Envoy proxies to forward access logs to SentryFlow.

> **Note**: Configuring telemetry could require some time to be fully implemented throughout the entire cluster.

To ensure that the pods in the `default` namespace are operational, execute the following command:

```bash
kubectl get pods -n default

NAME                       READY   STATUS    RESTARTS   AGE
httpbin-545f698b64-ncvq9   2/2     Running   0          44s
sleep-75bbc86479-fmf4p     2/2     Running   0          35s
```

## Step 2. Sending API Requests

Going forward, the `sleep` pod will initiate API requests to the `httpbin` service, which can be done using the following command:

```bash
export SOURCE_POD=$(kubectl get pod -l app=sleep -o jsonpath={.items..metadata.name})
kubectl exec "$SOURCE_POD" -c sleep -- curl -sS -v httpbin:8000/status/418
```

## Step 3. Checking Logs

There are two methods of checking logs with SentryFlow clients.

### 1. Log Client

To examine the logs exported by SentryFlow, you can use the following command:

```bash
kubectl logs -n sentryflow -l app=log-client

YYYY/MM/DD 17:03:37 [gRPC] Successfully connected to sentryflow.sentryflow.svc.cluster.local:8080
YYYY/MM/DD 17:40:28 [Client] Received log: timeStamp:"[YYYY/MM/DDT17:40:27.225Z]"  id:1707929670787152  srcNamespace:"default"  srcName:"sleep-75bbc86479-fmf4p"  srcLabel:{key:"app"  value:"sleep"}  srcLabel:{key:"pod-template-hash"  value:"75bbc86479"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"sleep"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"10.244.140.11"  srcPort:"44126"  srcType:"Pod"  dstNamespace:"default"  dstName:"httpbin"  dstLabel:{key:"app"  value:"httpbin"}  dstLabel:{key:"service"  value:"httpbin"}  dstIP:"10.105.103.198"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/1.1"  method:"GET"  path:"/status/418"  responseCode:418
YYYY/MM/DD 17:40:29 [Client] Received log: timeStamp:"[YYYY/MM/DDT17:40:28.845Z]"  id:1707929670787154  srcNamespace:"default"  srcName:"sleep-75bbc86479-fmf4p"  srcLabel:{key:"app"  value:"sleep"}  srcLabel:{key:"pod-template-hash"  value:"75bbc86479"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"sleep"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"10.244.140.11"  srcPort:"44158"  srcType:"Pod"  dstNamespace:"default"  dstName:"httpbin"  dstLabel:{key:"app"  value:"httpbin"}  dstLabel:{key:"service"  value:"httpbin"}  dstIP:"10.105.103.198"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/1.1"  method:"GET"  path:"/status/418"  responseCode:418
```

As expected, we should be able to observe the `/status/418` API request being made from the `sleep` pod to the `httpbin` service.

### 2. MongoDB Client

To inspect the data stored in MongoDB by SentryFlow, you can use the following command:

```bash
export MONGODB_POD=$(kubectl get pod -n sentryflow -l app=mongodb -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $MONGODB_POD -n sentryflow mongosh
```

Initiating this command will launch an interactive shell that can be used to explore the contents stored within the database. To examine the data in the database, refer to the subsequent commands provided.

```
test> use sentryflow;
switched to db sentryflow
sentryflow> db["APILogs"].find()
[
  {
    _id: ObjectId('65ccfa872b80bf0cec7dab83'),
    timestamp: '[YYYY-MM-DDT17:38:14.330Z]',
    id: Long('1707929670787151'),
    srcnamespace: 'default',
    srcname: 'sleep-75bbc86479-fmf4p',
    srclabel: {
      app: 'sleep',
      'pod-template-hash': '75bbc86479',
      'security.istio.io/tlsMode': 'istio',
      'service.istio.io/canonical-name': 'sleep',
      'service.istio.io/canonical-revision': 'latest'
    },
    srcip: '10.244.140.11',
    srcport: '47996',
    srctype: 'Pod',
    dstnamespace: 'default',
    dstname: 'httpbin',
    dstlabel: { app: 'httpbin', service: 'httpbin' },
    dstip: '10.105.103.198',
    dstport: '8000',
    dsttype: 'Service',
    protocol: 'HTTP/1.1',
    method: 'GET',
    path: '/status/418',
    responsecode: Long('418')
  }
]
```
