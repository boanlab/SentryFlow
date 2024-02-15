# Development Guide

SentryFlow operates within an Istio environment on Kubernetes, indicating that contributors to our project will need to have an Istio environment set up.

To minimize the hassle of installing and uninstalling Kubernetes and configuring Istio solely for our project, we have provided a straightforward Vagrantfile. This Vagrantfile initializes an Ubuntu virtual machine equipped with a fully operational Kubernetes and Istio environment.

## 1. Prerequisites

We employ Vagrant to provision VirtualBox virtual machines, creating a Kubernetes environment. As such, it is highly recommended to install the following package versions in your local environment:

- **[Vagrant](https://www.vagrantup.com/)** - v2.2.9
- **[VirtualBox](https://www.virtualbox.org/)** - v6.1

## 2. Starting up a VM

We have configured a Vagrantfile that initiates an Ubuntu 22.04 machine with Kubernetes pre-installed. The setup for Kubernetes is as described below:

> **Note:** Although Kubernetes officially advises the use of containerd over Docker as the Container Runtime Interface (CRI), we have chosen to use Docker as the CRI within our Kubernetes setup. This decision facilitates the building and testing of SentryFlow and its client images.

- Kubernetes: 1.23
- [CRI] Docker: 24.0.7
- [CNI] Calico: 0.3.1

To proceed, execute the following command within the `contribution/` directory:

```bash
$ vagrant up
Bringing machine 'sentryflow' up with 'virtualbox' provider...
==> sentryflow: Importing base box 'generic/ubuntu2204'...
==> sentryflow: Matching MAC address for NAT networking...
==> sentryflow: Checking if box 'generic/ubuntu2204' version '4.3.10' is up to date...
...
    sentryflow: clusterrolebinding.rbac.authorization.k8s.io/calico-node created
    sentryflow: clusterrolebinding.rbac.authorization.k8s.io/calico-cni-plugin created
    sentryflow: daemonset.apps/calico-node created
    sentryflow: deployment.apps/calico-kube-controllers created
```

This command will initiate the installation of the necessary development environment. The duration of this process may vary, primarily depending on the speed of your network connection, and could take several minutes to complete.

## 3. Development and Code Quality

### Development

After Vagrant has been successfully initialized, you can access the Istio and Kubernetes environment by executing the following steps:

```
$ vagrant ssh
```

The source code for SentryFlow will be located in `/home/vagrant/sentryflow` within the virtual environment, and this directory will also be synchronized with the current work directory on the host machine.

After making modifications to the source code of SentryFlow, you can build the changes by moving to the `sentryflow` directory and running the Makefile.

```
make build
```

Executing the Makefile will result in the construction of container images, each tagged as specified.

### Code Quality

To maintain a clean and secure code base for SentryFlow, we conduct several checks, including `gofmt` for code formatting, `golint` for code style and linting, and `gosec` for security scanning.

To evaluate the quality of your code, navigate to the `sentryflow` directory and execute the following commands:

```
make golint # will run golint checks
make gofmt # will run gofmt checks
make gosec # will run gosec checks
```

### Pull Request

Once everything is correctly set up, you are ready to create a pull request. Please refer to our guidelines for submitting PRs.

## 4. Cleaning Up

If you have successfully made changes to SentryFlow and wish to clean up the created workspace, you can simply use the following command:

```
$ vagrant destroy
    sentryflow: Are you sure you want to destroy the 'sentryflow' VM? [y/N] y
==> sentryflow: Forcing shutdown of VM...
==> sentryflow: Destroying VM and associated drives...
```

Executing the command will terminate and remove the VM that you were working on.
