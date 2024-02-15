# Nephio - OAI
This example is for capturing access logs from [Nephio](https://github.com/nephio-project/nephio)'s OAI Demo which is running on top of Istio using Numbat.

> **Note**: Be advised that any information regarding Nephio in this document might be outdated as Nephio is under early stage of development

## Step 1. Setting Up Nephio and Istio
In this document, we are going to discuss monitoring `oai-core` from the `core` cluster to monitor API activities within the control plane.

> **Note**: To setup Nephio, please refer to their official document for OAI [here](https://github.com/nephio-project/docs/blob/main/content/en/docs/guides/user-guides/exercise-2-oai.md), also, in this document, we are going to assume that every step through **Step 5** is performed properly.

Check if Nephio has cluster `core` running properly as well as `oai-core` namespaces.

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

To collect access logs within the namespace, we need to install Istio into the cluster.  
```
$ istioctl install --set profile=default --context core-admin@core
This will install the Istio 1.20.2 "default" profile (with components: Istio core, Istiod, and Ingress gateways) into the cluster. Proceed? (y/N) y
✔ Istio core installed                                                                                                                                                                                             
✔ Istiod installed                                                                                                                                                                                                 
✔ Ingress gateways installed                                                                                                                                                                                       
✔ Installation complete    
Made this installation the default for injection and validation.
```
Once Istio was successfully installed into the cluster, you can check if the Istio system is up and running properly by
```
$ kubectl get pods -n istio-system --context core-admin@core
```

## Step 2. Injecting Sidecars into Nephio
Till now, we have installed Istio in the cluster where `edge` cluster is running. However, this does NOT imply that we have any sidecar proxies running alongside each pod. To properly inject sidecar into Nephio, we need to perform the following steps:
### 2.1 Lowering Restriction: podSecurityStandard
Nephio creates clusters for each clusters (ex: `core`, `edge`, `regional`) using **podSecurityContext**. By default, Nephio has the following standards:
- `enforce`: `baseline`
- `audit` and `warn`: `restricted`

Those securityContexts avoid using `NET_ADMIN` and `NET_RAW` capabilities which are [required](https://istio.io/latest/docs/ops/deployment/requirements/) for proper side-car injection for `istio-init`. Therefore, we need to explicitly set those profiles across all namespaces as `privileged` for injecting Istio properly.

We can achieve this by:
```
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/audit=privileged --context core-admin@core
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/enforce=privileged --context core-admin@core
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/warn=privileged --context core-admin@core
```

> **Note**: Setting `podSecurityStandard` using `kubectl edit cluster regional-admin@regional` will revert settings to default, thus, directly modify the namespace instead.

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
Now, we need to inject sidecars using Istio by tagging namespaces with label. For current demo, we are just going to tag `core-admin@core` namespaces.
```
$ kubectl label namespace oai-core istio-injection=enabled --overwrite --context core-admin@core
namespace/oai-core labeled
```

## Step 3. Deploying Numbat
Now, it is time for Nubmat to be deployed. We can achieve this by:
```
$ kubectl create -f ../../../deployments/numbat.yaml --context core-admin@core
namespace/numbat created
serviceaccount/sa-numbat created
clusterrole.rbac.authorization.k8s.io/cr-numbat created
clusterrolebinding.rbac.authorization.k8s.io/rb-numbat created
deployment.apps/numbat-collector created
service/numbat-collector created
```
Also, additionally, we can deploy exporters for Numbat as well by:
```
$ kubectl create -f ../../../deployments/exporters.yaml --context core-admin@core
deployment.apps/logger created
persistentvolume/mongodb-pv-new created
persistentvolumeclaim/mongodb-pvc-new created
deployment.apps/mongodb-deployment created
service/mongo created
deployment.apps/mongo-client created
```
Verify if Pods in Numbat are properly by:
```
$ kubectl get pods -n numbat --context regional-admin@regional
NAME                                  READY   STATUS    RESTARTS   AGE
logger-75695cd4d4-z6rns               1/1     Running   0          37s
mongo-client-67dfb6ffbb-4psdh         1/1     Running   0          37s
mongodb-deployment-575549748d-9n6lx   1/1     Running   0          37s
numbat-collector-5bf9f6987c-kmpgx     1/1     Running   0          60s
```
> **Note**: Namespace `numbat` will not have `istio-injection=enabled`. If we enable this, each OpenTelemetry export will be logged as access logs, causing too many logs to be captured.
> **Note**: Deploying `numbat-collector` will automatically update Istio's mesh config (`istio-system/istio`) to export access logs to our collector

## Step 4. Restarting Deployments
Till now we have:
- Setup Numbat
- Prepared Istio injection
- Lowered podSecurityStandard

However, this will not generate any logs yet. For Numbat to get access logs, we need to add `telemetry` as well as restart deployments under  oai-logging`.
> **Note**: Restarting deployments before applying telemetry will make side-cars not report access logs to our collector. Therefore, apply telemetry before restarting deployments.

We can add telemetry to watch `oai-logging` namespace by:
```
$ kubectl create -f ./telemetry.yaml --context core-admin@core
telemetry.telemetry.istio.io/oai-logging created
```
Now, we are going to restart all deployments under namespace `oai-logging` by:
> **Note**: This requires restarting deployments within the `oai-logging` namespace. If there are any jobs running 

```
$ kubectl rollout restart deployments -n oai-core --context core-admin@core 
deployment.apps/amf-core restarted 
deployment.apps/ausf-core restarted deployment.apps/mysql restarted 
deployment.apps/nrf-core restarted 
deployment.apps/smf-core restarted 
deployment.apps/udm-core restarted 
deployment.apps/udr-core restarted
```
Once rollout restart was issued, we can check if the Pods actually have side-cars by:
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
As we can see, each Pod has 2 containers instead of 1. (You can check if this is `istio-proxy` container by `kubectl describe`)

## Step 5. Checking Logs
From now on, `numbat-collector` will be receiving logs from each deployment. We can check how deployments within our `oai-core` namespace is communicating by two methods: logger and MongoDB.

### 5.1 Checking Logger
The deployment `logger` will receive logs from `numbat-collector` using our export format and display those logs out as stdout. We can check live logs by:

```
$ kubectl rollout restart deployments -n oai-core --context core-admin@core
```
This will show live logs such as:
```
$ kubectl logs -n numbat -l app=logger --context core-admin@core
2024/02/15 03:44:45 [gRPC] Successfully connected to numbat-collector.numbat.svc.cluster.local:8080
2024/02/15 03:45:30 [Client] Received log: timeStamp:"[2024-02-15T03:45:30.153Z]"  id:1707968675718909  srcNamespace:"oai-core"  srcName:"smf-core-57bbdf59c4-x4jnk"  srcLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  srcLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  srcLabel:{key:"pod-template-hash"  value:"57bbdf59c4"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"smf-core"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcLabel:{key:"workload.nephio.org/oai"  value:"smf"}  srcIP:"192.168.1.57"  srcPort:"42954"  srcType:"Pod"  dstNamespace:"oai-core"  dstName:"nrf-core-5c74f7cdb4-mrk4w"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  dstLabel:{key:"pod-template-hash"  value:"5c74f7cdb4"}  dstLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  dstLabel:{key:"service.istio.io/canonical-name"  value:"nrf-core"}  dstLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  dstLabel:{key:"workload.nephio.org/oai"  value:"nrf"}  dstIP:"192.168.1.55"  dstPort:"80"  dstType:"Pod"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-nfm/v1/nf-instances?nf-type=NRF"  responseCode:503
2024/02/15 03:45:30 [Client] Received log: timeStamp:"[2024-02-15T03:45:30.732Z]"  id:1707968675718911  srcNamespace:"oai-core"  srcName:"udm-core-85c5478b94-bm4mv"  srcLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  srcLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  srcLabel:{key:"pod-template-hash"  value:"85c5478b94"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"udm-core"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcLabel:{key:"workload.nephio.org/oai"  value:"udm"}  srcIP:"192.168.1.54"  srcPort:"48406"  srcType:"Pod"  dstNamespace:"oai-core"  dstName:"nrf-core-5c74f7cdb4-mrk4w"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1alpha1"}  dstLabel:{key:"pod-template-hash"  value:"5c74f7cdb4"}  dstLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  dstLabel:{key:"service.istio.io/canonical-name"  value:"nrf-core"}  dstLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  dstLabel:{key:"workload.nephio.org/oai"  value:"nrf"}  dstIP:"192.168.1.55"  dstPort:"80"  dstType:"Pod"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-nfm/v1/nf-instances?nf-type=NRF"  responseCode:503
```

### 5.2 Checking MongoDB
Alongside with logger, we also have another exporter that stores every data coming from `numbat-collector` into MongoDB deployment. We can use `mongosh` for checking what is being stored in MongoDB. This can be achieved by:

```
$ export MONGODB_POD=$(kubectl get pod -n numbat -l app=mongodb --context core-admin@core -o jsonpath='{.items[0].metadata.name}')
$ kubectl exec -it $MONGODB_POD -n numbat --context core-admin@core mongosh
```
Once we have entered `mongosh` we can check entries stored in the DB. Numbat uses DB named `numbat` and collection `access-logs` for storing access logs.

An example command of checking all access logs stored in DB would be:
```
test> use numbat
use numbat
numbat> db["access-logs"].find()
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
Another example will be filtering out only `"protocol":"HTTP/1.1"` to check only API calls:

```
numbat> db["access-logs"].find({"protocol":"HTTP/2"})
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
