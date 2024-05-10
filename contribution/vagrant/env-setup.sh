#!/bin/bash

# == Build Essential == #

# update repo
sudo apt-get update

# install build-essential
sudo apt-get install -y build-essential

# == Containerd == #

# add GPG key
sudo apt-get install -y curl ca-certificates gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# add docker repository
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# update the docker repo
sudo apt-get update

# install containerd
sudo apt-get install -y containerd.io

# set up the default config file
sudo mkdir -p /etc/containerd
sudo containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i "s/SystemdCgroup = false/SystemdCgroup = true/g" /etc/containerd/config.toml
sudo systemctl restart containerd

# # == Kubernetes == #

# install k3s
curl -sfL https://get.k3s.io | K3S_KUBECONFIG_MODE="644" INSTALL_K3S_EXEC="--disable=traefik" sh -

echo "wait for initialization"
sleep 15

runtime="15 minute"
endtime=$(date -ud "$runtime" +%s)

while [[ $(date -u +%s) -le $endtime ]]
do
    status=$(kubectl get pods -A -o jsonpath={.items[*].status.phase})
    [[ $(echo $status | grep -v Running | wc -l) -eq 0 ]] && break
    echo "wait for initialization"
    sleep 1
done

# make kubectl accessable for vagrant user
mkdir -p /home/vagrant/.kube
sudo cp /etc/rancher/k3s/k3s.yaml /home/vagrant/.kube/config
sudo chown -R vagrant:vagrant /home/vagrant/.kube
echo "export KUBECONFIG=/home/vagrant/.kube/config" | tee -a /home/vagrant/.bashrc
PATH=$PATH:/bin:/usr/bin:/usr/local/bin

# == Istio == #

# move to home
cd /home/vagrant

# download istio
curl -L https://istio.io/downloadIstio | sh -

# copy istioctl to /usr/local/bin
sudo cp /home/vagrant/istio-*/bin/istioctl /usr/local/bin

# change permissions
sudo chown -R vagrant:vagrant /home/vagrant/istio-*

# install istio
su - vagrant -c "istioctl install --set profile=default -y"

# == Docker == #

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
sudo usermod -aG docker vagrant

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
