# Nephio - Free5GC

This example demonstrates capturing access logs from [Nephio](https://github.com/nephio-project/nephio), which operates on top of Istio using SentryFlow for log collection.

> **Note**: The information about Nephio provided in this document may be outdated, as Nephio is currently in the early stages of development.

## Step 1. Setting Up Nephio and Istio

In this document, we will discuss monitoring `free5gc-cp` from the `regional` cluster to observe API activities within the control plane.

> **Note**: To configure Nephio, please consult their official documentation available [here](https://github.com/nephio-project/docs/blob/main/content/en/docs/guides/user-guides/exercise-1-free5gc.md). Additionally, for the purpose of this document, it will be assumed that all steps up to and including **Step 6** have been executed correctly.

Ensure that the Nephio `regional` cluster is functioning correctly, as well as the `free5gc-cp` namespaces within it.

```bash
$ kubectl get pods --context regional-admin@regional -n free5gc-cp
NAME                            READY   STATUS    RESTARTS   AGE
free5gc-ausf-69569f564b-7ttn5   1/1     Running   0          16s
free5gc-nrf-5978f8f797-xkhnl    1/1     Running   0          16s
free5gc-nssf-697b486564-gtpm5   1/1     Running   0          16s
free5gc-pcf-55d6c758bb-rhsm5    1/1     Running   0          16s
free5gc-udm-78464dcd7b-j6s7n    1/1     Running   0          16s
free5gc-udr-565445b596-7c6zw    1/1     Running   0          16s
free5gc-webui-ddd948585-nzkrf   1/1     Running   0          16s
mongodb-0                       1/1     Running   0          7d9h
```

To gather access logs from within the namespace, Istio must be installed in the cluster. 

```
$ istioctl install --set profile=default --context regional-admin@regional
This will install the Istio 1.20.2 "default" profile (with components: Istio core, Istiod, and Ingress gateways) into the cluster. Proceed? (y/N) y
✔ Istio core installed                                                                                                                                                                                             
✔ Istiod installed                                                                                                                                                                                                 
✔ Ingress gateways installed                                                                                                                                                                                       
✔ Installation complete    
Made this installation the default for injection and validation.
```

After successfully installing Istio in the cluster, you can verify that the Istio system is operational and running correctly by executing the following command:

```
$ kubectl get pods -n istio-system --context regional-admin@regional
```

## Step 2. Injecting Sidecars into Nephio

Up to this point, Istio has been installed in the cluster where the `regional` cluster is operational. However, this does not necessarily mean that sidecar proxies are running alongside each pod. To ensure proper injection of sidecars into Nephio, the following steps need to be undertaken:

### 2.1 Lowering Restriction: podSecurityStandard

Nephio creates clusters for each type (e.g., `regional`, `edge01`, `edge02`) using **podSecurityContext**. By default, Nephio adheres to the following standards:

- `enforce`: `baseline`
- `audit` and `warn`: `restricted`

The security contexts employed by Nephio intentionally exclude the `NET_ADMIN` and `NET_RAW` capabilities, which are [required](https://istio.io/latest/docs/ops/deployment/requirements/) for the correct injection of the `istio-init` sidecar. Consequently, it is essential to explicitly designate these profiles as `privileged` across all namespaces to ensure Istio is injected properly.

We can achieve this by:

```
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/audit=privileged --context regional-admin@regional
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/enforce=privileged --context regional-admin@regional
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/warn=privileged --context regional-admin@regional
```

> **Note**: Modifying `podSecurityStandard` via `kubectl edit cluster regional-admin@regional` will reset the settings to their defaults. Therefore, it's recommended to directly alter the namespace configuration instead.

Now, verify if those labels were set properly by:

```
$ kubectl describe ns free5gc-cp --context regional-admin@regional
Name:         free5gc-cp
Labels:       app.kubernetes.io/managed-by=configmanagement.gke.io
              configsync.gke.io/declared-version=v1
              kubernetes.io/metadata.name=free5gc-cp
              pod-security.kubernetes.io/audit=privileged
              pod-security.kubernetes.io/enforce=privileged
              pod-security.kubernetes.io/warn=privileged
...
```

### 2.2 Preparing Sidecars

To inject sidecars using Istio, we will label the namespaces accordingly. For the purposes of this demonstration, we will specifically label the `free5gc-cp` namespaces.

```
$ kubectl label namespace free5gc-cp istio-injection=enabled --overwrite --context regional-admin@regional
namespace/free5gc-cp labeled
```

## Step 3. Deploying SentryFlow

Now is the moment to deploy SentryFlow. This can be accomplished by executing the following steps:

```
$ kubectl create -f ../../../deployments/sentryflow.yaml --context regional-admin@regional
namespace/sentryflow created
serviceaccount/sa-sentryflow created
clusterrole.rbac.authorization.k8s.io/cr-sentryflow created
clusterrolebinding.rbac.authorization.k8s.io/rb-sentryflow created
deployment.apps/sentryflow created
service/sentryflow created
```

Also, we can deploy exporters for SentryFlow by following these additional steps:

```
$ kubectl create -f ../../../deployments/log-client.yaml --context regional-admin@regional
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

However, this action alone will not yet produce any logs. To enable Numbat to collect access logs, it is necessary to add `telemetry` configurations and also restart the deployments under `free5gc-cp`.

> **Note**: Restarting deployments before implementing telemetry will result in the sidecars not transmitting access logs to our collector. Hence, it is important to apply telemetry configurations prior to restarting the deployments.

Telemetry can be configured to monitor the `free5gc-cp` namespace by executing the following steps:

```
$ kubectl create -f telemetry.yaml --context regional-admin@regional
telemetry.telemetry.istio.io/free5gc-logging created
```

To restart all deployments within the `free5gc-cp` namespace, you can proceed with the following command:

> **Note**: Restarting deployments within the `free5gc-cp` namespace is necessary. If there are any jobs currently running, additional steps may be needed to manage those jobs during the restart process. 

```
$ kubectl rollout restart deployment -n free5gc-cp --context regional-admin@regional
deployment.apps/free5gc-ausf restarted
deployment.apps/free5gc-nrf restarted
deployment.apps/free5gc-nssf restarted
deployment.apps/free5gc-pcf restarted
deployment.apps/free5gc-udm restarted
deployment.apps/free5gc-udr restarted
deployment.apps/free5gc-webui restarted
```

After issuing the rollout restart command, you can verify whether the Pods now include sidecars by executing the following command:

```
$ kubectl get pods --context regional-admin@regional -n free5gc-cp
NAME                            READY   STATUS    RESTARTS   AGE
free5gc-ausf-7d56c5f8db-bk54f   2/2     Running   0          21s
free5gc-nrf-7f7db5c645-kxfrc    2/2     Running   0          21s
free5gc-nssf-5477f65b9b-kfmbt   2/2     Running   0          21s
free5gc-pcf-c7b8ff6bb-t2zrq     2/2     Running   0          21s
free5gc-udm-65947bb776-xs6vf    2/2     Running   0          21s
free5gc-udr-67f5fdf44d-4ckwd    2/2     Running   0          21s
free5gc-webui-cf788755c-9bwzd   2/2     Running   0          21s
mongodb-0                       1/1     Running   0          7d10h
```

Observing that each Pod now contains 2 containers instead of just 1 indicates the presence of sidecars. To confirm that the additional container is indeed the `istio-proxy`, you can use the `kubectl describe` command for further verification.

## Step 5. Checking Logs

Starting from this point, `sentryflow` will begin receiving logs from each deployment. To examine how deployments within the `free5gc-cp` namespace are communicating, there are two methods available: using a log client and a mongo client.

### 5.1 Checking Logger

The `log-client` deployment is configured to receive logs from `sentryflow` in our specified export format and output these logs as stdout. To view live logs, you can use the following command:

```
$ kubectl logs -n sentryflow -l app=log-client -f --context regional-admin@regional
```

This will show live logs such as:

```
2024/02/12 20:37:19 [Client] Received log: timeStamp:"[2024-02-12T20:37:19.318Z]"  id:1707769691204491  srcNamespace:"free5gc-cp"  srcName:"free5gc-pcf-c7b8ff6bb-t2zrq"  srcLabel:{key:"nf"  value:"pcf"}  srcLabel:{key:"pod-template-hash"  value:"c7b8ff6bb"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-pcf"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.122"  srcPort:"45542"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-disc/v1/nf-instances?requester-nf-type=PCF&service-names=nudr-dr&target-nf-type=UDR"  responseCode:200
2024/02/12 20:37:20 [Client] Received log: timeStamp:"[2024-02-12T20:37:20.292Z]"  id:1707769691204493  srcNamespace:"free5gc-cp"  srcName:"free5gc-udm-65947bb776-xs6vf"  srcLabel:{key:"nf"  value:"udm"}  srcLabel:{key:"pod-template-hash"  value:"65947bb776"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-udm"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.124"  srcPort:"36488"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"PUT"  path:"/nnrf-nfm/v1/nf-instances/8ac564d2-e5cc-421c-96cc-8c57b9c85ded"  responseCode:201
2024/02/12 20:37:23 [Client] Received log: timeStamp:"[2024-02-12T20:37:23.594Z]"  id:1707769691204495  srcNamespace:"free5gc-cp"  srcName:"free5gc-ausf-7d56c5f8db-bk54f"  srcLabel:{key:"nf"  value:"ausf"}  srcLabel:{key:"pod-template-hash"  value:"7d56c5f8db"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-ausf"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.126"  srcPort:"35258"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"PUT"  path:"/nnrf-nfm/v1/nf-instances/9e1ddaeb-898f-4504-a247-b4a78b329a74"  responseCode:201
```

### 5.2 Checking MongoDB

We have another client (`mongo-client`) that stores all data received from the `sentryflow` into the MongoDB deployment. You can use `mongosh` to inspect the contents stored in MongoDB by executing the following command:

```
$ export MONGODB_POD=$(kubectl get pod -n sentryflow -l app=mongodb --context regional-admin@regional -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $MONGODB_POD -n sentryflow --context regional-admin@regional mongosh
```

Once we have entered `mongosh` we can check entries stored in the DB. SentryFlow uses DB named `sentryflow` and collection `access-logs` for storing access logs.

An example command of checking all access logs stored in DB would be:

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

Another example would involve filtering out only logs with `protocol":"HTTP/1.1` to specifically examine API calls:

```
sentryflow> db["access-logs"].find({"protocol":"HTTP/1.1"})
...
  {
    _id: ObjectId('65ca77e4ef0f86784e2fa545'),
    timestamp: '[2024-02-12T19:56:19.350Z]',
    id: Long('1707767512691241'),
    srcnamespace: 'free5gc-cp',
    srcname: 'free5gc-nssf-566df8589f-4wwt9',
    srclabel: {
      'security.istio.io/tlsMode': 'istio',
      'service.istio.io/canonical-name': 'free5gc-nssf',
      'service.istio.io/canonical-revision': 'latest',
      nf: 'nssf',
      'pod-template-hash': '566df8589f',
      project: 'free5gc'
    },
    srcip: '192.168.1.105',
    srcport: '45888',
    srctype: 'Pod',
    dstnamespace: 'free5gc-cp',
    dstname: 'free5gc-nrf-6f6484c6cb-cpnzk',
    dstlabel: {
      nf: 'nrf',
      'pod-template-hash': '6f6484c6cb',
      project: 'free5gc',
      'security.istio.io/tlsMode': 'istio',
      'service.istio.io/canonical-name': 'free5gc-nrf',
      'service.istio.io/canonical-revision': 'latest'
    },
    dstip: '192.168.1.94',
    dstport: '8000',
    dsttype: 'Pod',
    protocol: 'HTTP/1.1',
    method: 'PUT',
    path: '/nnrf-nfm/v1/nf-instances/99608079-71a4-48cd-9e0c-be0837655d2f',
    responsecode: Long('201')
...
```
