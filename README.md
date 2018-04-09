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
- [Continuous Integration](#continuous-integration)
- [Running locally](#running-locally)
  - [Testing webhooks](#testing-webhooks)

<!-- /MarkdownTOC -->

<a id="what-is-it"></a>
# What is it?

fabric8-jenkins-proxy (Jenkins Proxy) is the sister project to [fabric8-jenkins-idler](https://github.com/fabric8-services/fabric8-jenkins-idler)(Jenkins Idler).
Its task is to run a HTTP Proxy which sits in between an [openshift.io](https://openshift.io) user and its Jenkins instance within [openshift.io](https://openshift.io).
For more information refer to the Idler [README](https://github.com/fabric8-services/fabric8-jenkins-idler/blob/master/README.md).

![Architectural Diagram](https://camo.githubusercontent.com/0761536bd1260ce502604e4d2ff2592a79f56485/68747470733a2f2f646f63732e676f6f676c652e636f6d2f64726177696e67732f642f652f32504143582d31765268743172674e45533636663732395155634e356f475378745453475667554c5f38725f632d4b5f4a722d694b304657654844616b354933326c31794d69592d744e2d6e715168495259766f31472f7075623f773d34323626683d343431)

<a id="data-flow-diagrams"></a>
# Data flow diagrams

The following diagrams describe the data flow within the proxy for a received GitHub webhook respectively a direct user interaction with the Jenkins service:

[![GitHub Webhook](./docs/github-webhook.png)](./docs/github-webhook.png)

[![Jenkins UI](./docs/jenkins-ui.png)](./docs/jenkins-ui.png)

<a id="how-to-build"></a>
# How to build?

The following paragraphs describe how to build and work with the source.

<a id="prerequisites"></a>
## Prerequisites

The project is written in [Go](https://golang.org/), so you will need a working Go installation (Go version >= 1.9.1).

The build itself is driven by GNU [Make](https://www.gnu.org/software/make/) which also needs to be installed on your systems.

Last but not least, you need a running Docker daemon, since the final build artifact is a Docker container.

<a id="make-usage"></a>
## Make usage

<a id="compile-the-code"></a>
### Compile the code

   $ make build

<a id="build-the-container-image"></a>
### Build the container image

   $ make image

<a id="run-the-tests"></a>
### Run the tests

   $ make test

<a id="format-the-code"></a>
### Format the code

   $ make fmt

<a id="check-commit-message-format"></a>
### Check commit message format

   $ make validate_commits

<a id="clean-up"></a>
### Clean up

   $ make clean

More help is provided by `make help`.

<a id="dependency-management"></a>
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

<a id="continuous-integration"></a>
# Continuous Integration

At the moment Travis CI and CentOS CI are configured.
Both CI systems build all merges to master as well as pull requests.

| CI System |   |
|-----------|---|
| CentOS CI | [master](https://ci.centos.org/job/devtools-fabric8-jenkins-proxy-build-master/), [pr](https://ci.centos.org/job/devtools-fabric8-jenkins-proxy/)|
| Travis CI | [master](https://travis-ci.org/fabric8-services/fabric8-jenkins-proxy/), [pr](https://travis-ci.org/fabric8-services/fabric8-jenkins-proxy/pull_requests)|

<a id="running-locally"></a>
# Running locally

The repository contains a script [`setupLocalProxy.sh`](./scripts/setupLocalProxy.sh) which can be used to run the Proxy locally.
A prerequisite for this is access to https://console.rh-idev.openshift.com/.
To run the script you need to export your OpenShift access token for console.rh-idev.openshift.com as DSAAS_PREVIEW_TOKEN.
Note, In order to port forward you need to edit permissions on the dsaas-preview namespace.
You need to have jq installed to run these commands. For fedora use `sudo dnf install jq`

Usage: ./scripts/setupLocalProxy.sh [start|stop|env|unset]

This script is used to run the Jenkins Proxy on localhost.
As a prerequisite OPENSHIFT_API_TOKEN needs to be exported.
In your shell (from the root of fabric8-jenkins-proxy):

To start proxy and other required services
```
[user@localhost ~]$ export DSAAS_PREVIEW_TOKEN=<dsaas-preview-token> 
[user@localhost ~]$ ./scripts/setupLocalProxy.sh start 
```
Run command below this in a seperate terminal so that we can see logs of the above command. 
```
[user@localhost ~]$ export DSAAS_PREVIEW_TOKEN=<dsaas-preview-token> 
[user@localhost ~]$ eval $(./scripts/setupLocalProxy.sh env) 
[user@localhost ~]$ fabric8-jenkins-proxy 
```

After you stop `fabric8-jenkins-proxy`, you would want to stop all the dependency services as well
To remove postgres container and stop port-forwarding to prod-preview's idler service and tenant service
```
[user@localhost ~]$ ./scripts/setupLocalProxy.sh stop
```

Services running as a part of this local setup:
  - idler on 9001
  - tenant service on 9002
  - postgres on 5432
  
<a id="testing-webhooks"></a>
## Testing webhooks

You can trigger local webhook delivery like so.
Go to a GitHub repository generated by the OpenShift.io launcher.
Find the webhook settings under Settings->Webhooks.
There you can see the recent deliveries.
Copy the payload of a webhook delivery into a file `webhook-payload.json`.
Then execute the following curl command:

    $ curl http://localhost:8080/github-webhook/ \
    -H "Content-Type: application/json" \
    -H "User-Agent: GitHub-Hookshot/c494ff1" \
    -H "X-GitHub-Event: status" \
    -d @webhook-payload.json


<a id="testing-through-ui"></a>
## Testing Through UI

Any request that is made to proxy(i.e., port 8080) regardless of the path, proxy will send a request to idler to unidle jenkins, if it is idled.

    curl http://localhost:8080/*

This would show a spinning wheel until jenkins is idle. On running locally the html page might not exist so, it will show a message on not finding the html page.

<a id="apis"></a>
## APIs

This project opens two ports 9091 and 8080. Proxy runs on 8080 and API router runs 9091. 
The API router has only one API, which is info API. An example is as follows

    Request: GET http://localhost:9091/api/info/ksagathi-preview

    Response: {"namespace":"ksagathi-preview","requests":0,"last_visit":0,"last_request":0}
    


