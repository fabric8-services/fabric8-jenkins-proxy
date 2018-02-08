# GitHub webhook migration

<!-- MarkdownTOC -->

- [What is it?](#what-is-it)
- [Build container](#build-container)
- [Run locally](#run-locally)
- [Run migration](#run-migration)

<!-- /MarkdownTOC -->


<a name="what-is-it"></a>
# What is it?

This directory contains script and Dockerfile for generating a container to run the GitHub webhook migration as required for the Jenkins Proxy to rollout to all users.

See also https://github.com/fabric8-services/fabric8-jenkins-proxy/issues/94

<a name="build-container"></a>
# Build container

    $ docker build -t fabric8-jenkins-proxy/webhook-migration .

<a name="run-locally"></a>
# Run locally

    $ docker run -e OPENSHIFT_URL=<URL of OpenShift cluster> \
    -e OPENSHIFT_API_TOKEN=<OpenShift service account token> \
    -e AUTH_URL=<URL of auth service> \
    -e TENANT_URL=<URL of tentant service> \
    -e AUTH_TOKEN=<Auth token for tentant service> \
    -e JENKINS_OLD_URL_SUFFIX=<Suffix of the old webhook URL. Used to match hook to update> \
    -e JENKINS_PROXY_URL=<URL of Jenskins proxy> \
    -e PRIVATE_KEY=<Private key for OSIO token generation> \
    -e TARGET_ENV=<stage|prod> \
    -e DRY_RUN=<true|false> \
    fabric8-jenkins-proxy/webhook-migration

The tenant service can be port forwarded to your local host:

    $ oc port-forward <tentant-pod> <local-port>:8080

If you then want to run the script as Docker container, you will need to forward this local port to a interface/IP which is reachable from within the container.
OpenShift's port forwarding only binds to the loopback interface.
You can use `ncat` for that (assuming you are using 9090 for the oc port forward and 9091 for the incontainer port):

    $ ncat -k -l $(ipconfig getifaddr en0) 9091 --sh-exec "ncat localhost 9090"

<a name="run-migration"></a>
# Run migration

    # Dry run against staging and specifying a single namespave
    $ oc process -p TARGET_ENV=stage -p DRY_RUN=true -p CHECK_NAMESPACES=johndoe -f webhook-migration.job.yaml | oc apply -f -

    # Dry run against staging
    $ oc process -p TARGET_ENV=stage -p DRY_RUN=true -f webhook-migration.job.yaml | oc apply -f -
    
    # Apply changes against prod
    $ oc process -p TARGET_ENV=prod -p DRY_RUN=false -f webhook-migration.job.yaml | oc apply -f -
    
Do delete the job and start over:

    $ oc delete job/webhook-migration


