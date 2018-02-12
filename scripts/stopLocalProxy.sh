#!/usr/bin/env bash
#
# Used to stop a locally running Proxy.

pid=$(pgrep fabric8-jenkins-proxy)
if [ -n "${pid}" ]; then
    kill -15 ${pid}
fi