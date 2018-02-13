# fabric8-jenkins-proxy [![Build Status](https://ci.centos.org/buildStatus/icon?job=devtools-fabric8-jenkins-proxy-build-master)](https://ci.centos.org/view/Devtools/job/devtools-fabric8-jenkins-proxy-build-master/) [![Build Status](https://travis-ci.org/fabric8-services/fabric8-jenkins-proxy.svg?branch=master)](https://travis-ci.org/fabric8-services/fabric8-jenkins-proxy.svg?branch=master)

<!-- MarkdownTOC -->

- [What is it?](#what-is-it)
- [Data flow diagrams](#data-flow-diagrams)
- [How to build?](#how-to-build)
  - [Prerequisites](#prerequisites)
  - [Make usage](#make-usage)
    - [Compile the code](#compile-the-code)
    - [Build the container image](#build-the-container-image)
    - [Run the tests](#run-the-tests)
    - [Format the code](#format-the-code)
    - [Check commit message format](#check-commit-message-format)
    - [Clean up](#clean-up)
  - [Dependency management](#dependency-management)

<!-- /MarkdownTOC -->


<a name="what-is-it"></a>
# What is it?

fabric8-jenkins-proxy (Jenkins Proxy) is the sister project to [fabric8-jenkins-idler](https://github.com/fabric8-services/fabric8-jenkins-idler)(Jenkins Idler).
Its task is to run a HTTP Proxy which sits in between an [openshift.io](https://openshift.io) user and its Jenkins instance within [openshift.io](https://openshift.io).
For more information refer to the Idler [README](https://github.com/fabric8-services/fabric8-jenkins-idler/blob/master/README.md).

![Architectural Diagram](https://camo.githubusercontent.com/0761536bd1260ce502604e4d2ff2592a79f56485/68747470733a2f2f646f63732e676f6f676c652e636f6d2f64726177696e67732f642f652f32504143582d31765268743172674e45533636663732395155634e356f475378745453475667554c5f38725f632d4b5f4a722d694b304657654844616b354933326c31794d69592d744e2d6e715168495259766f31472f7075623f773d34323626683d343431)

<a name="data-flow-diagrams"></a>
# Data flow diagrams

The following diagrams describe the data flow within the proxy for a received GitHub webhook respectively a direct user interaction with the Jenkins service:

[![GitHub Webhook](./docs/github-webhook.png)](./docs/github-webhook.png)

[![Jenkins UI](./docs/jenkins-ui.png)](./docs/jenkins-ui.png)

<a name="how-to-build"></a>
# How to build?

The following paragraphs describe how to build and work with the source.

<a name="prerequisites"></a>
## Prerequisites

The project is written in [Go](https://golang.org/), so you will need a working Go installation (Go version >= 1.9.1).

The build itself is driven by GNU [Make](https://www.gnu.org/software/make/) which also needs to be installed on your systems.

Last but not least, you need a running Docker daemon, since the final build artifact is a Docker container.

<a name="make-usage"></a>
## Make usage

<a name="compile-the-code"></a>
### Compile the code

   $ make build

<a name="build-the-container-image"></a>
### Build the container image

   $ make image

<a name="run-the-tests"></a>
### Run the tests

   $ make test

<a name="format-the-code"></a>
### Format the code

   $ make fmt

<a name="check-commit-message-format"></a>
### Check commit message format

   $ make validate_commits

<a name="clean-up"></a>
### Clean up

   $ make clean

More help is provided by `make help`.

<a name="dependency-management"></a>
## Dependency management

The dependencies of the project are managed by [Dep](https://github.com/golang/dep).
To add or change the current dependencies you need to delete the Dep lock file (_Gopkg.lock_), update the dependency list (_Gopkg.toml_) and then regenerate the lock file.
The process looks like this:

    $ make clean
    $ rm Gopkg.lock
    # Update Gopkg.toml with the changes to the dependencies
    $ make build
    $ git add Gopkg.toml Gopkg.lock
    $ git commit






