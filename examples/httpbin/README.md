# Single HTTP Requests
> **Note**: This requires Numbat and Istio on top of Kubernetes environment set up properly. If you have not met the requirements, please refer to the [getting started](../../docs/getting_started.md) document for more information.

This document demonstrates how Numbat can capture API logs for simple HTTP requests. The demo utilizes Istio's example with `sleep` and `httpbin`. 

Please refer to the [Istio's official document](https://istio.io/latest/docs/tasks/observability/logs/otel-provider/) for more information. The deployments `sleep` and `httpbin` is required to be set up properly as well as setting the default namespace as [Istio injection enabled](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection).

## Step 1. Verify Services
Verify if everything with Istio is set up properly. First, we shall check if the `default` namespace has Istio injection enabled. This can be achieved by the following command:

```
$ kubectl describe namespace default
Name:         default
Labels:       istio-injection=enabled
...
```
If the namespace `default` has label `istio-injection=enabled`, this was set properly. Now, apply the `telemetry.yaml` in this directory by following command:

```
$ kubectl create -f telemetry.yaml
telemetry.telemetry.istio.io/sleep-logging created
```

This will setup `telemetry` for Istio which will tell Envoy proxies to send access log to our Numbat collector.


> **Note**: Setting up telemetry might take some time to be applied across the whole cluster. 

Also, verify if the pods in the namespace `default` are up and running by:
```
$ kubectl get pods -n default
NAME                       READY   STATUS    RESTARTS   AGE
httpbin-545f698b64-ncvq9   2/2     Running   0          44s
sleep-75bbc86479-fmf4p     2/2     Running   0          35s
```

## Step 2. Sending API Requests
From now on, the `sleep` pod will send API requests to the `httpbin` service. This can be achieved by the following command:
```
$ export SOURCE_POD=$(kubectl get pod -l app=sleep -o jsonpath={.items..metadata.name})
$ kubectl exec "$SOURCE_POD" -c sleep -- curl -sS -v httpbin:8000/status/418
```

## Step 3. Checking Logs
There are two methods of checking logs with exporters.

### 1. Logger
We can check logs being exported from Numbat by following the command:
```
$ kubectl logs -n numbat -l app=logger
2024/02/14 17:03:37 [gRPC] Successfully connected to numbat-collector.numbat.svc.cluster.local:8080
2024/02/14 17:40:28 [Client] Received log: timeStamp:"[2024-02-14T17:40:27.225Z]"  id:1707929670787152  srcNamespace:"default"  srcName:"sleep-75bbc86479-fmf4p"  srcLabel:{key:"app"  value:"sleep"}  srcLabel:{key:"pod-template-hash"  value:"75bbc86479"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"sleep"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"10.244.140.11"  srcPort:"44126"  srcType:"Pod"  dstNamespace:"default"  dstName:"httpbin"  dstLabel:{key:"app"  value:"httpbin"}  dstLabel:{key:"service"  value:"httpbin"}  dstIP:"10.105.103.198"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/1.1"  method:"GET"  path:"/status/418"  responseCode:418
2024/02/14 17:40:29 [Client] Received log: timeStamp:"[2024-02-14T17:40:28.845Z]"  id:1707929670787154  srcNamespace:"default"  srcName:"sleep-75bbc86479-fmf4p"  srcLabel:{key:"app"  value:"sleep"}  srcLabel:{key:"pod-template-hash"  value:"75bbc86479"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"sleep"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"10.244.140.11"  srcPort:"44158"  srcType:"Pod"  dstNamespace:"default"  dstName:"httpbin"  dstLabel:{key:"app"  value:"httpbin"}  dstLabel:{key:"service"  value:"httpbin"}  dstIP:"10.105.103.198"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/1.1"  method:"GET"  path:"/status/418"  responseCode:418
```

As we have expected,  we can observe `/status/418` API request from `sleep` to `httpbin` service. 

### 2. MongoDB
We can check what is stored inside the MongoDB from Numbat by the following command:
```
$ export MONGODB_POD=$(kubectl get pod -n numbat -l app=mongodb -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $MONGODB_POD -n numbat mongosh
```
This will start up an interactive shell which we can use for observing what is stored inside the database. Refer to the following commands to check data stored in DB.

```
test> use numbat;
switched to db numbat
numbat> db["access-logs"].find()
[
  {
    _id: ObjectId('65ccfa872b80bf0cec7dab83'),
    timestamp: '[2024-02-14T17:38:14.330Z]',
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