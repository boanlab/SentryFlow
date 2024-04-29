#!/bin/bash

# == Build Essential == #

# update repo
sudo apt-get update

# install build-essential
sudo apt-get install -y build-essential

# == Containerd == #

# update repo
sudo apt-get update

# install curl
sudo apt-get install -y curl

# add GPG key
sudo apt-get install -y ca-certificates gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# add Docker repository
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# update the Docker repo
sudo apt-get update

# install containerd
sudo apt-get install -y containerd.io

# set up the default config file
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i "s/SystemdCgroup = false/SystemdCgroup = true/g" /etc/containerd/config.toml
sudo systemctl restart containerd

# == Kubernetes == #

# update repo
sudo apt-get update

# install curl and apt-transport-https
sudo apt-get install -y curl apt-transport-https ca-certificates gpg

# add the key for kubernetes repo
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

# add sources.list.d
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list

# update repo
sudo apt-get update

# install the latest version
sudo apt-get install -y kubeadm kubelet kubectl
sudo apt-mark hold kubelet kubeadm kubectl

# disable Swap
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# initialize Kubernetes
sudo kubeadm init --pod-network-cidr=10.244.0.0/16 | tee -a /home/vagrant/k8s_init.log

# disable master isolation
kubectl taint nodes --all node-role.kubernetes.io/master-
kubectl taint nodes --all node-role.kubernetes.io/control-plane-

# wait for a while
sleep 5

# install Calico
kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.4/manifests/calico.yaml

# make kubectl accessable for vagrant user
sudo mkdir -p /home/vagrant/.kube
sudo cp -i /etc/kubernetes/admin.conf /home/vagrant/.kube/config
sudo chown $(id -u vagrant):$(id -g vagrant) /home/vagrant/.kube/config

# == Istio == #

# move to home
cd /home/vagrant

# download istio
curl -L https://istio.io/downloadIstio | sh -

# copy istioctl to /usr/local/bin
sudo cp $HOME/istio-*/bin/istioctl /usr/local/bin

# install istio
istioctl install --set profile=default -y

# == Docker == #

# update repo
sudo apt-get update

# add GPG key
sudo apt-get install -y curl ca-certificates gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# add Docker repository
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# update the Docker repo
sudo apt-get update

# install Docker
sudo apt-get install -y docker-ce && sleep 5

# configure daemon.json
sudo mkdir -p /etc/docker
cat <<EOF | sudo tee /etc/docker/daemon.json
{
    "exec-opts": ["native.cgroupdriver=systemd"],
    "log-driver": "json-file",
    "log-opts": {
        "max-size": "100m"
    },
    "storage-driver": "overlay2"
}
EOF

# start Docker
sudo systemctl restart docker && sleep 5

# add user to docker
sudo usermod -aG docker $USER

# bypass to run docker command
sudo chmod 666 /var/run/docker.sock

# == Go == #

# update repo
sudo apt-get update

# install wget
sudo apt -y install wget

# instsall golang
goBinary=$(curl -s https://go.dev/dl/ | grep linux | head -n 1 | cut -d'"' -f4 | cut -d"/" -f3)
wget https://dl.google.com/go/$goBinary -O /tmp/$goBinary
sudo tar -C /usr/local -xvzf /tmp/$goBinary
rm /tmp/$goBinary

# add GOPATH, GOROOT
echo >> /home/vagrant/.bashrc
echo "export GOPATH=\$HOME/go" >> /home/vagrant/.bashrc
echo "export GOROOT=/usr/local/go" >> /home/vagrant/.bashrc
echo "export PATH=\$PATH:/usr/local/go/bin:\$HOME/go/bin" >> /home/vagrant/.bashrc
echo >> /home/vagrant/.bashrc

# create a directory for Go
mkdir -p /home/vagrant/go
chown -R vagrant:vagrant /home/vagrant/go
