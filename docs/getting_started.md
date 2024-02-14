# Getting Started
This guide walks through steps to easily deploy Numbat system on top of Istio for collecting API logs. Each steps includes the commands required and their respective descriptions.

> **Note**: Nubmat is in its early stage of development. Any information mentioned here might be outdated or subject to change without any notice.

## 0. Prerequisites
Numbat operates on top of the Istio environment. The following table shows the environment in which Numbat was successfully deployed and checked running properly.

|System Name|Version|
|--|--|
|Ubuntu|22.04, 20.04|
|[Istio](https://istio.io/latest/)|1.20.2|
|[Kubernetes](https://kubernetes.io/)|v1.27.1|

> **Note**: For details on Kubernetes settings, such as CNI, CRI, and their runtime settings, please refer to the [compatability matrix](k8s_compatibility.d) for more information.

## 1. Deploying Numbat Collector
Numbat can be deployed just using `kubectl` command. This can be achieved by the following commands:

```
$ git clone https://github.com/boanlab/numbat
$ cd numbat/
$ kubectl create -f deployments/numbat.yaml
namespace/numbat created
serviceaccount/sa-numbat created
clusterrole.rbac.authorization.k8s.io/cr-numbat created
clusterrolebinding.rbac.authorization.k8s.io/rb-numbat created
deployment.apps/numbat-collector created
service/numbat-collector created
```

This will setup a namespace named `numbat` and will setup K8s resources required. 
> **Note**: Numbat will automatically patch Istio's meshConfig to set up `extensionProviders` for log collection to Numbat collector.

Then check if Numbat Collector is up and running by:
```
$ kubectl get pods -n numbat
NAME                               READY   STATUS    RESTARTS   AGE
numbat-collector-cd95d79b4-9q7d7   1/1     Running   0          4m41s
```
## 2. Deploying Numbat Exporters
Until now, the Numbat collector is set up in the cluster. Numbat collector also exports the logs coming from Istio through its custom gRPC format. For more information on how this data is sent, please refer to the [Numbat exporter guide](numbat_exporter_guide.md) for more information.

For demonstration purposes, we have created two demo exporters
- `logger`: Simply log everything coming from Numbat exporter service
- `mongo-client`: Stores every logs coming from Numbat exporter service to a MongoDB service.

These exporters and MongoDB service can be deployed into the cluster under namespace `numbat` by following the command:

```
$ kubectl create -f deployments/exporters.yaml
deployment.apps/logger created
persistentvolume/mongodb-pv-new created
persistentvolumeclaim/mongodb-pvc-new created
deployment.apps/mongodb-deployment created
service/mongo created
deployment.apps/mongo-client created
```

> **Note**: MongoDB deployment `mongodb-deployment` uses PersistentVolume named `mongodb-pv-new` which is mounted to the host's path for storing logs permanently. This is set as `/home/boan/numbat` by default. Therefore if you would like to set it as another directory which fits into your environment, please change this accordingly in the `exporters.yaml` file.

Then check if those exporters and MongoDB are properly up and running by:
```
$ kubectl get pods -n numbat
NAME                                  READY   STATUS    RESTARTS   AGE
logger-6c8864655f-h2sdv               1/1     Running   0          5m28s
mongo-client-7cbf6b888f-vd69g         1/1     Running   0          5m28s
mongodb-deployment-6f5d9fc599-zwnxj   1/1     Running   0          5m28s
...
```

If you see `logger`, `mongo-client`, `mongodb-deployment` running in the namespace, everything is set up properly.  

## 3. Use Cases and Examples
Until now, we have successfully set up Numbat system into the Istio service mesh and Kubernetes cluster. For more information on use cases and examples, please refer to the following documents.

The following links are listed with their level of complexity, from basic to complex.
- [Single HTTP Requests](../examples/httpbin/README.md)
- [RobotShop Demo](../examples/robotshop/README.md)
- [Nephio Free5gc Workload](../examples/nephio/free5gc/README.md)
- [Nephio OAI Workload](../examples/nephio/oai/README.md)
