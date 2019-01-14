![alpha](https://img.shields.io/badge/stability%3F-beta-yellow.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubic-project/dex-operator)](https://goreportcard.com/report/github.com/kubic-project/dex-operator)
[![CircleCI](https://circleci.com/gh/kubic-project/dex-operator/tree/master.svg?style=svg)](https://circleci.com/gh/kubic-project/dex-operator/tree/master)

# Dex Operator:

A Dex operator for Kubernetes, developed inside the
[Kubic](https://en.opensuse.org/Portal:Kubic) project.


- [Features](#features)
- [Quickstart](docs/dex-operator.md)
- [Devel](docs/devel.md)
- [Additional Info](#extra)

## Features

* Automatic (re)configuration of a Dex instance with
some [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)s.
* Automatic deployment/re-deployment/stop of the Dex instance
depending on the current number of LDAP connectors.
* Automatic certificate management: the dex-operator will create
a certificate and get it signed from the API server for you.

# Devel 
* See the [development documentation](docs/devel.md) if you intend to contribute to this project.

# See also

* The [dex-operator image](https://hub.docker.com/r/opensuse/dex-operator/) in the Docker Hub.
* The [kubic-init](https://github.com/kubic-project/kubic-init) container, a container for
bootstrapping a Kubernetes cluster on top of [MicroOS](https://en.opensuse.org/Kubic:MicroOS)
(an openSUSE-Tumbleweed-based OS focused on running containers).
* The [Kubic Project](https://en.opensuse.org/Portal:Kubic) home page.
