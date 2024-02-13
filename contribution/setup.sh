#!/bin/bash

# From our lab's tools scripts, install Docker and Kubeadn
git clone https://github.com/boanlab/tools.git
bash tools/containers/install-docker.sh
bash tools/kubernetes/install-kubeadm.sh

# Initialize Kubernetes for single node
# We are going to use Docker with Kubernetes and CNI with Calico.
# Even if Docker is outdated with Kubernetes, it is easier for us
# to build containers and deploy them without us having to export images
# into containerd. So for development purpose, we are using Docker as CRI.
export MULTI=false
export CNI=calico
sudo swapoff -a
bash tools/kubernetes/initialize-kubeadm.sh
bash tools/kubernetes/deploy-cni.sh

# Make kubectl related commands accessable for vagrant user
sudo mkdir -p /home/vagrant/.kube
sudo cp -i /etc/kubernetes/admin.conf /home/vagrant/.kube/config
sudo chown $(id -u vagrant):$(id -g vagrant) /home/vagrant/.kube/config

# Till here, we have successfully installed Kubernetes in Vagrant
# Now install Istio
sudo apt-get install make
curl -L https://istio.io/downloadIstio | sh -
export PATH="$PATH:/home/vagrant/istio-1.20.3/bin"
istioctl install --set profile=default -y

# Now install golang, this is for golint, gosec, gofmt
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

