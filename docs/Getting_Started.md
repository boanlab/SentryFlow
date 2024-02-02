# Getting Started

## Build Images
- Build Numbat(Collector) image
```
cd numbat
make
```

- Build Exporter images
```
cd exporters/client-mongo
make
```
```
cd exporters/client-stdout
make
```

## Set up Exporter and Numbat
- Create a namespace where Numbat will be created
```
kubectl apply -f /deployments/0-setup-ns.yaml
```

- Set up exporter examples
```
kubectl apply -f /deployments/1-setup-our-exporters.yaml
```

- Set up Numbat(Collector)
```
kubectl apply -f /deployments/2-setup-collector.yaml
```

> 1.Modify istio-system/istio ConfigMap include our collector
>
> 2.Add labels to namespaces for Istio injection
>
> 3.Restart all deployments in those namespaces
>
> This patching adds automatically


## Make Numbat Pod
```
```
