# Nephio
This example is for capturing access logs from [Nephio](https://github.com/nephio-project/nephio) which is running on top of Istio using Numbat.

> **Note**: Be advised that any information regarding Nephio in this document might be outdated as Nephio is under early stage of development

## Step 1. Setting Up Nephio and Istio
In this document, we are going to discuss monitoring `free5gc-cp` from the `regional` cluster to monitor API activities within the control plane.

> **Note**: To setup Nephio, please refer to their official document [here](https://github.com/nephio-project/docs/blob/main/content/en/docs/guides/user-guides/exercise-1-free5gc.md), also, in this document, we are going to assume that every step through **Step 6** is performed properly.


Check if Nephio has cluster `regional` running properly as well as `free5gc-cp` namespaces.

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

To collect access logs within the namespace, we need to install Istio into the cluster.  
```
$ istioctl install --set profile=default --context regional-admin@regional
This will install the Istio 1.20.2 "default" profile (with components: Istio core, Istiod, and Ingress gateways) into the cluster. Proceed? (y/N) y
✔ Istio core installed                                                                                                                                                                                             
✔ Istiod installed                                                                                                                                                                                                 
✔ Ingress gateways installed                                                                                                                                                                                       
✔ Installation complete    
Made this installation the default for injection and validation.
```
Once Istio was successfully installed into the cluster, you can check if the Istio system is up and running properly by
```
$ kubectl get pods -n istio-system --context regional-admin@regional
```

## Step 2. Injecting Sidecars into Nephio
Till now, we have installed Istio in the cluster where `regional` cluster is running. However, this does NOT imply that we have any sidecar proxies running alongside each pod. To properly inject sidecar into Nephio, we need to perform the following steps:
### 2.1 Lowering Restriction: podSecurityStandard
Nephio creates clusters for each clusters (ex: `regional`, `edge01`, `edge02`) using **podSecurityContext**. By default, Nephio has the following standards:
- `enforce`: `baseline`
- `audit` and `warn`: `restricted`

Those securityContexts avoid using `NET_ADMIN` and `NET_RAW` capabilities which are [required](https://istio.io/latest/docs/ops/deployment/requirements/) for proper side-car injection for `istio-init`. Therefore, we need to explicitly set those profiles across all namespaces as `privileged` for injecting Istio properly.

We can achieve this by:
```
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/audit=privileged --context regional-admin@regional
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/enforce=privileged --context regional-admin@regional
$ kubectl label --overwrite ns --all pod-security.kubernetes.io/warn=privileged --context regional-admin@regional
```

> **Note**: Setting `podSecurityStandard` using `kubectl edit cluster regional-admin@regional` will revert settings to default, thus, directly modify the namespace instead.

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
Now, we need to inject sidecars using Istio by tagging namespaces with label. For current demo, we are just going to tag `free5gc-cp` namespaces.
```
$ kubectl label namespace free5gc-cp istio-injection=enabled --overwrite --context regional-admin@regional
namespace/free5gc-cp labeled
```

## Step 3. Deploying Numbat
Now, it is time for Nubmat to be deployed. We can achieve this by:
```
$ kubectl create -f ../../deployments/numbat.yaml --context regional-admin@regional
namespace/numbat created
serviceaccount/sa-numbat created
clusterrole.rbac.authorization.k8s.io/cr-numbat created
clusterrolebinding.rbac.authorization.k8s.io/rb-numbat created
deployment.apps/numbat-collector created
service/numbat-collector created
```
Also, additionally, we can deploy exporters for Numbat as well by:
```
$ kubectl create -f ../../deployments/exporters.yaml --context regional-admin@regional
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

However, this will not generate any logs yet. For Numbat to get access logs, we need to add `telemetry` as well as restart deployments under `free5gc-cp`.
> **Note**: Restarting deployments before applying telemetry will make side-cars not report access logs to our collector. Therefore, apply telemetry before restarting deployments.


We can add telemetry to watch `free5gc-cp` namespace by:
```
$ kubectl create -f telemetry.yaml --context regional-admin@regional
telemetry.telemetry.istio.io/free5gc-logging created
```
Now, we are going to restart all deployments under namespace `free5gc-cp` by:
> **Note**: This requires restarting deployments within the `free5gc-cp` namespace. If there are any jobs running 

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
Once rollout restart was issued, we can check if the Pods actually have side-cars by:
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
As we can see, each Pod has 2 containers instead of 1. (You can check if this is `istio-proxy` container by `kubectl describe`)

## Step 5. Checking Logs
From now on, `numbat-collector` will be receiving logs from each deployment. We can check how deployments within our `free5gc-cp` namespace is communicating by two methods: logger and MongoDB.

### 5.1 Checking Logger
The deployment `logger` will receive logs from `numbat-collector` using our export format and display those logs out as stdout. We can check live logs by:

```
$ kubectl logs -n numbat -l app=logger -f --context regional-admin@regional
```
This will show live logs such as:
```
2024/02/12 20:37:19 [Client] Received log: timeStamp:"[2024-02-12T20:37:19.318Z]"  id:1707769691204491  srcNamespace:"free5gc-cp"  srcName:"free5gc-pcf-c7b8ff6bb-t2zrq"  srcLabel:{key:"nf"  value:"pcf"}  srcLabel:{key:"pod-template-hash"  value:"c7b8ff6bb"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-pcf"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.122"  srcPort:"45542"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"GET"  path:"/nnrf-disc/v1/nf-instances?requester-nf-type=PCF&service-names=nudr-dr&target-nf-type=UDR"  responseCode:200
2024/02/12 20:37:20 [Client] Received log: timeStamp:"[2024-02-12T20:37:20.292Z]"  id:1707769691204493  srcNamespace:"free5gc-cp"  srcName:"free5gc-udm-65947bb776-xs6vf"  srcLabel:{key:"nf"  value:"udm"}  srcLabel:{key:"pod-template-hash"  value:"65947bb776"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-udm"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.124"  srcPort:"36488"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"PUT"  path:"/nnrf-nfm/v1/nf-instances/8ac564d2-e5cc-421c-96cc-8c57b9c85ded"  responseCode:201
2024/02/12 20:37:23 [Client] Received log: timeStamp:"[2024-02-12T20:37:23.594Z]"  id:1707769691204495  srcNamespace:"free5gc-cp"  srcName:"free5gc-ausf-7d56c5f8db-bk54f"  srcLabel:{key:"nf"  value:"ausf"}  srcLabel:{key:"pod-template-hash"  value:"7d56c5f8db"}  srcLabel:{key:"project"  value:"free5gc"}  srcLabel:{key:"security.istio.io/tlsMode"  value:"istio"}  srcLabel:{key:"service.istio.io/canonical-name"  value:"free5gc-ausf"}  srcLabel:{key:"service.istio.io/canonical-revision"  value:"latest"}  srcIP:"192.168.1.126"  srcPort:"35258"  srcType:"Pod"  dstNamespace:"free5gc-cp"  dstName:"nrf-nnrf"  dstLabel:{key:"app.kubernetes.io/managed-by"  value:"configmanagement.gke.io"}  dstLabel:{key:"app.kubernetes.io/version"  value:"v3.1.1"}  dstLabel:{key:"configsync.gke.io/declared-version"  value:"v1"}  dstLabel:{key:"nf"  value:"nrf"}  dstLabel:{key:"project"  value:"free5gc"}  dstIP:"10.141.104.225"  dstPort:"8000"  dstType:"Service"  protocol:"HTTP/2"  method:"PUT"  path:"/nnrf-nfm/v1/nf-instances/9e1ddaeb-898f-4504-a247-b4a78b329a74"  responseCode:201
```

### 5.2 Checking MongoDB
Alongside with logger, we also have another exporter that stores every data coming from `numbat-collector` into MongoDB deployment. We can use `mongosh` for checking what is being stored in MongoDB. This can be achieved by:

```
$ export MONGODB_POD=$(kubectl get pod -n numbat -l app=mongodb --context regional-admin@regional --template '{{range .items}}{{.metadata.name}}{{"\n"}}{
{end}}')
$ kubectl exec -it $MONGODB_POD -n numbat --context regional-admin@regional mongosh
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
numbat> db["access-logs"].find({"protocol":"HTTP/1.1"})
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
