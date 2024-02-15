# Nephio - OAI


This example demonstrates how to capture access logs from the [Nephio](https://github.com/nephio-project/nephio)'s OAI Demo, which operates on top of Istio, utilizing SentryFlow for log capture.

> **Note**: The information about Nephio provided in this document may be outdated, as Nephio is currently in the early stages of development.

## Step 1. Setting Up Nephio and Istio

In this document, we will discuss how to monitor the `oai-core` component within the `core` cluster to observe API activities in the control plane.

> **Note**: To set up Nephio, please consult the official OAI documentation available [here](https://github.com/nephio-project/docs/blob/main/content/en/docs/guides/user-guides/exercise-2-oai.md). For the purposes of this document, it will be assumed that all steps up to and including **Step 5** have been executed correctly.

Ensure that the Nephio `core` cluster is functioning correctly, as well as the `oai-core` namespaces within it.

```bash
$ kubectl get pods -n oai-core --context core-admin@core
NAME                        READY   STATUS    RESTARTS   AGE
amf-core-56c68b7487-g2clh   1/1     Running   0          10h
ausf-core-7885cb865-hd9pz   1/1     Running   0          10h
mysql-7dd4cc6945-pj6xz      1/1     Running   0          10h
nrf-core-d4f69557d-wptds    1/1     Running   0          10h
smf-core-59bcf4576c-t6rwr   1/1     Running   0          10h
udm-core-c7d67cb4d-r4zwn    1/1     Running   0          10h
udr-core-69c56bcbd5-whjb9   1/1     Running   0          10h
```

To gather access logs from within the namespace, Istio must be installed in the cluster.

```
$ istioctl install --set profile=default --context core-admin@core
This will install the Istio 1.20.2 "default" profile (with components: Istio core, Istiod, and Ingress gateways) into the cluster. Proceed? (y/N) y
✔ Istio core installed                                                                                                                                                                                             
✔ Istiod installed                                                                                                                                                                                                 
✔ Ingress gateways installed                                                                                                                                                                                       
✔ Installation complete    
Made this installation the default for injection and validation.
```

After successfully installing Istio in the cluster, you can verify that the Istio system is operational and running correctly by executing the following command:

```
$ kubectl get pods -n istio-system --context core-admin@core
```

## Step 2. Injecting Sidecars into Nephio

Up to this point, Istio has been installed in the cluster where the `edge` cluster is operational. However, this does not necessarily mean that sidecar proxies are running alongside each pod. To ensure proper injection of sidecars into Nephio, the following steps need to be undertaken:

### 2.1 Lowering Restriction: podSecurityStandard

Nephio creates clusters for each type (e.g., `core`, `edge`, `regional`) using **podSecurityContext**. By default, Nephio adheres to the following standards:

- `enforce`: `baseline`
- `audit` and `warn`: `restricted`

The security contexts employed by Nephio intentionally exclude the `NET_ADMIN` and `NET_RAW` capabilities, which are [required](https://istio.io/latest/docs/ops/deployment/requirements/) for the correct injection of the `istio-init` sidecar. Consequently, it is essential to explicitly designate these profiles as `privileged` across all namespaces to ensure Istio is injected properly.

We can achieve this by:

```
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/audit=privileged --context core-admin@core
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/enforce=privileged --context core-admin@core
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/warn=privileged --context core-admin@core
```

> **Note**: Modifying `podSecurityStandard` via `kubectl edit cluster regional-admin@regional` will reset the settings to their defaults. Therefore, it's recommended to directly alter the namespace configuration instead.

Now, verify if those labels were set properly by:

```
$ kubectl describe ns oai-core --context core-admin@core
Name:         oai-core
Labels:       app.kubernetes.io/managed-by=configmanagement.gke.io
              configsync.gke.io/declared-version=v1
              kubernetes.io/metadata.name=oai-core
              pod-security.kubernetes.io/audit=privileged
              pod-security.kubernetes.io/enforce=privileged
              pod-security.kubernetes.io/warn=privileged
...
```

### 2.2 Preparing Sidecars

To inject sidecars using Istio, we will label the namespaces accordingly. For the purposes of this demonstration, we will specifically label the `core-admin@core` namespaces.

```
$ kubectl label namespace oai-core istio-injection=enabled --overwrite --context core-admin@core
namespace/oai-core labeled
```

## Step 3. Deploying SentryFlow

Now is the moment to deploy SentryFlow. This can be accomplished by executing the following steps:

```
$ kubectl create -f ../../../deployments/sentryflow.yaml --context core-admin@core
namespace/sentryflow created
serviceaccount/sa-sentryflow created
clusterrole.rbac.authorization.k8s.io/cr-sentryflow created
clusterrolebinding.rbac.authorization.k8s.io/rb-sentryflow created
deployment.apps/sentryflow created
service/sentryflow created
```

Also, we can deploy exporters for SentryFlow by following these additional steps:

```
$ kubectl create -f ../../../deployments/log-client.yaml --context core-admin@core
deployment.apps/log-client created

$ kubectl create -f ../../../deployments/mongo-client.yaml --context regional-admin@regional
deployment.apps/mongodb created
service/mongodb created
deployment.apps/mongo-client created
```

Verify if Pods in SentryFlow are properly by:

```
$ kubectl get pods -n sentryflow --context regional-admin@regional
NAME                            READY   STATUS    RESTARTS   AGE
log-client-75695cd4d4-z6rns     1/1     Running   0          37s
mongo-client-67dfb6ffbb-4psdh   1/1     Running   0          37s
mongodb-575549748d-9n6lx        1/1     Running   0          37s
sentryflow-5bf9f6987c-kmpgx     1/1     Running   0          60s
```

> **Note**: 
The `sentryflow` namespace will not have `istio-injection=enabled`. Enabling this would result in each OpenTelemetry export being logged as an access log, leading to an excessive number of logs being captured.

> **Note**: Deploying `sentryflow` will automatically modify the Istio mesh configuration (`istio-system/istio`) to direct the export of access logs to it.

## Step 4. Restarting Deployments

Till now we have:
- Setup SentryFlow
- Prepared Istio injection
- Lowered podSecurityStandard

However, this action alone will not yet produce any logs. To enable Numbat to collect access logs, it is necessary to add `telemetry` configurations and also restart the deployments under `oai-logging`.

> **Note**: Restarting deployments before implementing telemetry will result in the sidecars not transmitting access logs to our collector. Hence, it is important to apply telemetry configurations prior to restarting the deployments.

Telemetry can be configured to monitor the `oai-logging` namespace by executing the following steps:

```
$ kubectl create -f ./telemetry.yaml --context core-admin@core
telemetry.telemetry.istio.io/oai-logging created
```

To restart all deployments within the `oai-logging` namespace, you can proceed with the following command:

> **Note**: Restarting deployments within the `oai-logging` namespace is necessary. If there are any jobs currently running, additional steps may be needed to manage those jobs during the restart process. 

```
$ kubectl rollout restart deployments -n oai-core --context core-admin@core 
deployment.apps/amf-core restarted 
deployment.apps/ausf-core restarted deployment.apps/mysql restarted 
deployment.apps/nrf-core restarted 
deployment.apps/smf-core restarted 
deployment.apps/udm-core restarted 
deployment.apps/udr-core restarted
```

After issuing the rollout restart command, you can verify whether the Pods now include sidecars by executing the following command:

```
$ kubectl get pods -n oai-core --context core-admin@core
NAME                         READY   STATUS     RESTARTS   AGE
amf-core-76967858c4-w4mlt    2/2     Running    0          8m3s
ausf-core-6bfd5576c5-sprb4   2/2     Running    0          8m10s
mysql-764b8f5ff5-7hgcv       2/2     Running    0          8m2s
nrf-core-5c74f7cdb4-mrk4w    2/2     Running    0          8m10s
smf-core-57bbdf59c4-x4jnk    2/2     Running    0          8m5s
udm-core-85c5478b94-bm4mv    2/2     Running    0          8m10s
...
```

Observing that each Pod now contains 2 containers instead of just 1 indicates the presence of sidecars. To confirm that the additional container is indeed the `istio-proxy`, you can use the `kubectl describe` command for further verification.

## Step 5. Checking Logs

Starting from this point, `sentryflow` will begin receiving logs from each deployment. To examine how deployments within the `oai-core` namespace are communicating, there are two methods available: using a log client and a mongo client.

### 5.1 Checking Logs using a Log Client

The `log-client` deployment is configured to receive logs from `sentryflow` in our specified export format and output these logs as stdout. To view live logs, you can use the following command:

```
$ kubectl logs -n sentryflow -l app=log-client -f --context core-admin@core
```

This will show live logs such as:

```
2024/02/15 03:45:30 [Client] Received log: timeStamp:"[2024-02-15T03:45:30.153Z]"  id:1707968675718909  srcNamespace:"oai-core"  srcName:"smf-core-57bbdf59c4-x4jnk"  srcLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  srcLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  srcLabel:{key:"pod-template-hash"  value:"57bbdf59c4"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"smf-core"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcLabel:{key:"workload.nephio.org/oai"  value:"smf"}  srcIP:"192.168.1.57"  srcPort:"42954"  srcType:"Pod"  dstNamespace:"oai-core"  dstName:"nrf-core-5c74f7cdb4-mrk4w"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  dstLabel:{key:"pod-template-hash"  value:"5c74f7cdb4"}  dstLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  dstLabel:{key:"service.istio.io/canonical-name"  value:"nrf-core"}  dstLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  dstLabel:{key:"workload.nephio.org/oai"  value:"nrf"}  dstIP:"192.168.1.55"  dstPort:"80"  dstType:"Pod"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-nfm/v1/nf-instances?nf-type=NRF"  responseCode:503
2024/02/15 03:45:30 [Client] Received log: timeStamp:"[2024-02-15T03:45:30.732Z]"  id:1707968675718911  srcNamespace:"oai-core"  srcName:"udm-core-85c5478b94-bm4mv"  srcLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  srcLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  srcLabel:{key:"pod-template-hash"  value:"85c5478b94"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"udm-core"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcLabel:{key:"workload.nephio.org/oai"  value:"udm"}  srcIP:"192.168.1.54"  srcPort:"48406"  srcType:"Pod"  dstNamespace:"oai-core"  dstName:"nrf-core-5c74f7cdb4-mrk4w"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  dstLabel:{key:"pod-template-hash"  value:"5c74f7cdb4"}  dstLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  dstLabel:{key:"service.istio.io/canonical-name"  value:"nrf-core"}  dstLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  dstLabel:{key:"workload.nephio.org/oai"  value:"nrf"}  dstIP:"192.168.1.55"  dstPort:"80"  dstType:"Pod"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-nfm/v1/nf-instances?nf-type=NRF"  responseCode:503
```

### 5.2 Checking Logs in MongoDB


We have another client (`mongo-client`) that stores all data received from the `sentryflow` into the MongoDB deployment. You can use `mongosh` to inspect the contents stored in MongoDB by executing the following command:

```
$ export MONGODB_POD=$(kubectl get pod -n sentryflow -l app=mongodb --context core-admin@core -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $MONGODB_POD -n sentryflow --context core-admin@core mongosh
```

Once we have entered `mongosh` we can check entries stored in the DB. SentryFlow uses DB named `sentryflow` and collection `api-logs` for storing access logs.

An example command for retrieving all access logs stored in the database would be:

```
test> use sentryflow
use sentryflow
sentryflow> db["api-logs"].find()
...
  {
    _id: ObjectId('65ca77e4ef0f86784e2fa544'),
    timestamp: '[2024-02-12T19:56:19.298Z]',
    id: Long('1707767512691239'),
    srcnamespace: 'free5gc-cp',
    srcname: 'free5gc-nssf-566df8589f-4wwt9',
    srclabel: {
      'pod-template-hash': '566df8589f',
      project: 'free5gc',
      'security.istio.io/tlsMode': 'istio',
      'service.istio.io/canonical-name': 'free5gc-nssf',
      'service.istio.io/canonical-revision': 'latest',
      nf: 'nssf'
    },
    srcip: '192.168.1.105',
    srcport: '53008',
    srctype: 'Pod',
    dstnamespace: 'free5gc-cp',
    dstname: 'nrf-nnrf',
    dstlabel: {
      'app.kubernetes.io/managed-by': 'configmanagement.gke.io',
      'app.kubernetes.io/version': 'v3.1.1',
      'configsync.gke.io/declared-version': 'v1',
      nf: 'nrf',
      project: 'free5gc'
    },
    dstip: '10.141.104.225',
    dstport: '8000',
    dsttype: 'Service',
    protocol: 'HTTP/2',
    method: 'PUT',
    path: '/nnrf-nfm/v1/nf-instances/99608079-71a4-48cd-9e0c-be0837655d2f',
    responsecode: Long('201')
  },
...
```

Another example would involve filtering out only logs with `protocol":"HTTP/2` to specifically examine API calls:

```
sentryflow> db["api-logs"].find({"protocol":"HTTP/2"})
...
  {
    _id: ObjectId('65cd871bb9e996068ab49250'),
    timestamp: '[2024-02-15T03:38:02.636Z]',
    id: Long('1707968025200999'),
    srcnamespace: 'kube-system',
    srcname: 'kube-scheduler-core-cxfgb-gt8tr',
    srclabel: { component: 'kube-scheduler', tier: 'control-plane' },
    srcip: '172.18.0.5',
    srcport: '3479',
    srctype: 'Pod',
    dstnamespace: 'oai-core',
    dstname: 'nrf-core-696b59c448-4dn52',
    dstlabel: {
      'pod-template-hash': '696b59c448',
      'security.istio.io/tlsMode': 'istio',
      'service.istio.io/canonical-name': 'nrf-core',
      'service.istio.io/canonical-revision': 'latest',
      'workload.nephio.org/oai': 'nrf',
      'app.kubernetes.io/managed-by': 'configmanagement.gke.io',
      'configsync.gke.io/declared-version': 'v1alpha1'
    },
    dstip: '192.168.1.35',
    dstport: '80',
    dsttype: 'Pod',
    protocol: 'HTTP/2',
    method: 'PATCH',
    path: '/nnrf-nfm/v1/nf-instances/863bdd79-b36b-4c85-b6ed-61bfc1cb5de3',
    responsecode: Long('503')
  },
...
```
