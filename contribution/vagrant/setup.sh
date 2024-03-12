#!/bin/bash

# From BoanLab's tools scripts
git clone https://github.com/boanlab/tools.git

# Install Docker
bash tools/containers/install-containerd.sh

# Install Kubeadm
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.24/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.24/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo sysctl -w net.ipv4.ip_forward=1
sudo swapoff -a
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl

# Disable Swap
sudo swapoff -a

# Initialize Kubernetes for single node
export MULTI=false
bash tools/kubernetes/initialize-kubeadm.sh

# Deploy Calico
export CNI=calico
bash tools/kubernetes/deploy-cni.sh

# Make kubectl related commands accessable for vagrant user
sudo mkdir -p /home/vagrant/.kube
sudo cp -i /etc/kubernetes/admin.conf /home/vagrant/.kube/config
sudo chown $(id -u vagrant):$(id -g vagrant) /home/vagrant/.kube/config

# Now install Istio
sudo apt-get install make
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.20.3 sh -
export PATH="$PATH:/home/vagrant/istio-1.20.3/bin"
istioctl install --set profile=default -y
sudo chown -R vagrant /home/vagrant/istio-1.20.3/

# Now install golang, this is for golint, gosec, gofmt
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Setup bashrc
echo export GOPATH="/home/vagrant/go" >> /home/vagrant/.bashrc
echo export PATH="$PATH:/usr/local/go/bin:/home/vagrant/istio-1.20.3/bin:/home/vagrant/go/bin/" >> /home/vagrant/.bashrc
