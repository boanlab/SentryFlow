# Development Guide
Numbat runs on top of Istio environment with Kubernetes. This means that anybody who wants to contribute to our project would require an Istio environment installed.

To reduce the overhead of installing and uninstalling Kubernetes as well as Istio setup just for our project, we have set up a simple Vagantfile which starts up a Ubuntu virtual machine with a fully functioning Kubernetes with Istio environment in it.

## 1. Prerequisites
We utilize Vagrant for provisioning VirtualBox virtual machines to provide a Kubernetes environment. Therefore, the following package suggested version is highly recommended to be installed in your local environment:

- **[Vagrant](https://www.vagrantup.com/)** - v2.2.9
- **[VirtualBox](https://www.virtualbox.org/)** - v6.1

## 2. Starting up VM
We have set up a Vagrantfile that starts a Ubuntu22.04 machine with Kubernetes installed. The Kubernetes setup is as follows:
> **Note:** We understand that Kubernetes has officially recommended using containerd instead of Docker as CRI. However, using containerd as CRI for Kubernetes in our development environemtn will require us to export images built in Docker to containerd images every time. Therefore, to remove this extra step, we are using Docker as CRI for Kubernetes. 

- Kubernetes: 1.23
- [CRI] Docker: 24.0.7
- [CNI] Calico: 0.3.1

Execute following command under `contributing/` directory
```bash
$ vagrant up
Bringing machine 'numbat' up with 'virtualbox' provider...
==> numbat: Importing base box 'generic/ubuntu2204'...
==> numbat: Matching MAC address for NAT networking...
==> numbat: Checking if box 'generic/ubuntu2204' version '4.3.10' is up to date...
...
    numbat: clusterrolebinding.rbac.authorization.k8s.io/calico-node created
    numbat: clusterrolebinding.rbac.authorization.k8s.io/calico-cni-plugin created
    numbat: daemonset.apps/calico-node created
    numbat: deployment.apps/calico-kube-controllers created
```
This will start installing the required environment for development. Depending on your network connection, this might take some minutes.

## 3. Development and Code Quality
### Development
Once Vagrant has successfully been initialized, you can use the Istio and Kubernetes environment by:
```
$ vagrant ssh
```
Project source for Numbat will be stored under `/home/vagrant/numbat` and this will be synced with the current host's workdirectory as well. 

Once a change has been made to Numbat's source code, you can build it by navigating to `/numbat` directory and executing Makefile
```
make build
```
This will build container images with given tags.

### Code Quality
For Numbat to retain clean and safe code base, we perform some checks. Those include: gofmt, golint, and gosec.

You can check your code's quality by navigating to `/numbat` directory and executing following commands
```
make golint # will run golint checks
make gofmt # will run gofmt checks
make gosec # will run gosec checks
```

### Pull Request
Once everything was properly set, you can now pull request. Please refer to our guidelines for PR.

## 4. Cleaning Up
Once you have successfully made changes into Numbat and wish to clean up the workspace that has been created, you can simply use:
```
$ vagrant destroy
    numbat: Are you sure you want to destroy the 'numbat' VM? [y/N] y
==> numbat: Forcing shutdown of VM...
==> numbat: Destroying VM and associated drives...
```
This will destroy the VM that you were working on. The changes that you have made will be stored under `work/` directory.