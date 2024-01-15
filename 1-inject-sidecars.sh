#!/bin/bash

# Get namespaces using kubectl and exclude the headers
kubectl get ns --no-headers | awk '{ if ($1 != "kube-system" && $1 != "istio-system" && $1 != "metallb-system" ) print $1 }' | xargs -I {} kubectl label namespace {} istio-injection=enabled --overwrite && kubectl get deployment --all-namespaces -o custom-columns=:metadata.namespace,:metadata.name --no-headers | awk '{ if ($1 != "kube-system" && $1 != "istio-system" && $1 != "metallb-system" ) print "kubectl rollout restart deployment -n " $1 " " $2 }' | sh

