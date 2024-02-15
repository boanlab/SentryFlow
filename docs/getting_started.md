# Getting Started

This guide provides a step-by-step process for deploying SentryFlow on Istio, aimed at enhancing API observability and security. It includes detailed commands for each step along with their explanations.

> **Note**: SentryFlow is currently in the early stages of development. Please be aware that the information provided here may become outdated or change without notice.

## 1. Prerequisites

SentryFlow functions within the Istio framework. Below is a table detailing the environments where SentryFlow has been successfully deployed and verified to be operational.

|System Name|Version|
|--|--|
|Ubuntu|22.04, 20.04|
|[Istio](https://istio.io/latest/)|1.20.2|
|[Kubernetes](https://kubernetes.io/)|v1.27.1|

> **Note**: For information on Kubernetes configurations, including Container Network Interface (CNI), Container Runtime Interface (CRI), and their respective runtime settings, please consult the [compatability matrix](k8s_compatibility.md).

## 2. Deploying SentryFlow

SentryFlow can be deployed using `kubectl` command. The deployment can be accomplished with the following commands:

```
$ git clone https://github.com/5GSEC/sentryflow
$ cd sentryflow/
$ kubectl create -f deployments/sentryflow.yaml
namespace/sentryflow created
serviceaccount/sa-sentryflow created
clusterrole.rbac.authorization.k8s.io/cr-sentryflow created
clusterrolebinding.rbac.authorization.k8s.io/rb-sentyflow created
deployment.apps/sentryflow created
service/sentryflow created
```

This process will create a namespace named `sentryflow` and will establish the necessary Kubernetes resources. 

> **Note**: SentryFlow will automatically modify Istio's `meshConfig` to configure `extensionProviders`, facilitating SentryFlow's API log collection.

Then, check if SentryFlow is up and running by:

```
$ kubectl get pods -n sentryflow
NAME                         READY  STATUS    RESTARTS   AGE
sentryflow-cd95d79b4-9q7d7   1/1    Running   0          4m41s
```

## 3. Deploying SentryFlow Clients

SentryFlow has now been established within the cluster. In addition, SentryFlow exports API logs and metrics through gRPC. For further details on how this data is transmitted, please consult the [SentryFlow Client Guide](sentryflow_client_guide.md).

For testing purposes, two simple clients have been developed.

- `log-client`: Simply log everything coming from SentryFlow service
- `mongo-client`: Stores every logs coming from SentryFlow service to a MongoDB service.

These clients can be deployed into the cluster under namespace `sentryflow` by following the command:

- `log-client`
    ```
    $ kubectl create -f deployments/log-client.yaml
    deployment.apps/log-client created
    ```

- `mongo-client`
    ```
    $ kubectl create -f deployments/mongo-client.yaml
    deployment.apps/mongodb created
    service/mongodb created
    deployment.apps/mongo-client created
    ```

Then, check if those clients and MongoDB are properly up and running by:

```
$ kubectl get pods -n sentryflow
NAME                                  READY   STATUS    RESTARTS   AGE
log-client-6c8864655f-h2sdv           1/1     Running   0          5m28s
mongo-client-7cbf6b888f-vd69g         1/1     Running   0          5m28s
mongodb-6f5d9fc599-zwnxj              1/1     Running   0          5m28s
...
```

If you observe `log-client`, `mongo-client`, and `mongodb` running within the namespace, the setup has been completed successfully.

## 3. Use Cases and Examples

Up to this point, SentryFlow has been successfully integrated into the Istio service mesh and Kubernetes cluster. For additional details on use cases and examples, please consult the accompanying documentation.

The links below are organized by their level of complexity, starting from basic and progressing to more complex.

- [Single HTTP Requests](../examples/httpbin/README.md)
- [RobotShop Demo Microservice](../examples/robotshop/README.md)
- [Nephio Free5gc Workload](../examples/nephio/free5gc/README.md)
- [Nephio OAI Workload](../examples/nephio/oai/README.md)
