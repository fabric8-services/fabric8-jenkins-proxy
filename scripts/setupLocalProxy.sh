#!/usr/bin/env bash
#
# Used to run Proxy locally.
#

LOCAL_IDLER_PORT=${LOCAL_IDLER_PORT:-9001}
LOCAL_TENANT_PORT=${LOCAL_TENANT_PORT:-9002}
LOCAL_POSTGRES_PORT=${LOCAL_POSTGRES_PORT:-5432}

###############################################################################
# Prints help message
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
printHelp() {
    cat << EOF
Usage: ${0##*/} [start|stop]

This script is used to run the Jenkins Proxy on localhost.
As a prerequisite OPENSHIFT_API_TOKEN and  JC_AUTH_TOKEN need to be exported.
In your shell (from the root of fabric8-jenkins-proxy):

> export DSAAS_PREVIEW_TOKEN=<dsaas-preview token>
> export JC_OPENSHIFT_API_TOKEN=<OpenShift API token>
> export JC_AUTH_TOKEN=<auth token>
> ./scripts/${0##*/} start
> eval \$(./scripts/${0##*/} env)
> fabric8-jenkins-proxy
EOF
}

###############################################################################
# Forwards the jenkins-idler service to localhost
# Globals:
#   LOCAL_IDLER_PORT - local Idler port
# Arguments:
#   None
# Returns:
#   None
###############################################################################
forwardIdler() {
    pod=$(oc get pods -l deploymentconfig=jenkins-idler -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine Idler pod name"
        return
    fi
    port=$(oc get pods -l deploymentconfig=jenkins-idler -o json | jq -r '.items[0].spec.containers[0].ports[0].containerPort')

    if lsof -Pi :${LOCAL_IDLER_PORT} -sTCP:LISTEN -t >/dev/null ; then
        echo "INFO: Local Idler port ${LOCAL_IDLER_PORT} already listening. Skipping oc port-forward"
        return
    fi

    while :
    do
	    oc port-forward ${pod} ${LOCAL_IDLER_PORT}:${port}
	    echo "Idler port forward stopped with exit code $?.  Respawning.." >&2
	    sleep 1
    done
    echo "Idler port forward stopped." >&2
}

###############################################################################
# Forwards the f8tenant service to localhost
# Globals:
#   LOCAL_TENANT_PORT - local tenant port
# Arguments:
#   None
# Returns:
#   None
###############################################################################
forwardTenant() {
    pod=$(oc get pods -l deploymentconfig=f8tenant -o json | jq -r '.items[0].metadata.name')
    if [ "${pod}" == "null" ] ; then
        echo "WARN: Unable to determine Tenant pod name"
        return
    fi
    port=$(oc get pods -l deploymentconfig=f8tenant -o json | jq -r '.items[0].spec.containers[0].ports[0].containerPort')

    if lsof -Pi :${LOCAL_TENANT_PORT} -sTCP:LISTEN -t >/dev/null ; then
        echo "INFO: Local Tenant port ${LOCAL_TENANT_PORT} already listening. Skipping oc port-forward"
        return
    fi

    while :
    do
	    oc port-forward ${pod} ${LOCAL_TENANT_PORT}:${port}
	    echo "Tenant port forward stopped with exit code $?.  Respawning.." >&2
	    sleep 1
    done
    echo "Tenant port forward stopped." >&2
}

###############################################################################
# Runs a Postgres Docker container on port 5
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
runPostgres() {
    container=$(docker ps -q --filter "name=postgres")
    if [ -z "${container}" ] ; then
        docker run --name postgres -e POSTGRES_PASSWORD=postgres -d -p ${LOCAL_POSTGRES_PORT}:5432 postgres 1>&2
    fi
}

###############################################################################
# Starts the port forwarding as well as the Postgres Docker container.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
start() {
    [ -z "${DSAAS_PREVIEW_TOKEN}" ] && printHelp && exit 1
    [ -z "${OPENSHIFT_API_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_AUTH_TOKEN}" ] && printHelp && exit 1

    oc login https://api.rh-idev.openshift.com --token=${DSAAS_PREVIEW_TOKEN} > /dev/null

    forwardIdler &
    forwardTenant &
    runPostgres &
}

###############################################################################
# Displays the required environment settings for evaluation.
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
env() {
    [ -z "${JC_OPENSHIFT_API_TOKEN}" ] && printHelp && exit 1
    [ -z "${JC_AUTH_TOKEN}" ] && printHelp && exit 1

    echo export JC_OPENSHIFT_API_URL=https://api.free-stg.openshift.com
    echo export JC_OPENSHIFT_API_TOKEN=${JC_OPENSHIFT_API_TOKEN}
    echo export JC_KEYCLOAK_URL=https://sso.prod-preview.openshift.io
    echo export JC_WIT_API_URL=https://api.prod-preview.openshift.io
    echo export JC_REDIRECT_URL=https://jenkins.prod-preview.openshift.io
    echo export JC_AUTH_URL=https://auth.prod-preview.openshift.io
    echo export JC_POSTGRES_PORT=${LOCAL_POSTGRES_PORT}
    echo export JC_POSTGRES_HOST=localhost
    echo export JC_POSTGRES_PASSWORD=postgres
    echo export JC_POSTGRES_DATABASE=postgres
    echo export JC_AUTH_TOKEN=${JC_AUTH_TOKEN}
    echo export JC_IDLER_API_URL=http://localhost:${LOCAL_IDLER_PORT}
    echo export JC_F8TENANT_API_URL=http://localhost:${LOCAL_TENANT_PORT}
}


###############################################################################
# Stops oc-port forwarding and Docker container
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
###############################################################################
stop() {
    pids=$(pgrep -a -f -d " " "setupLocalProxy.sh start")
    pids+=$(pgrep -a -f -d " " "oc port-forward")
    kill -9 ${pids}
    docker rm -f postgres
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  env)
    env
    ;;
  *)
    printHelp
esac

