#!/bin/bash
#
# Uses Siege to run performance test of OpenShift.io Idler/Proxy
#
# Prerequisites:
# - Siege (https://www.joedog.org/siege-home/) installation
# - osio built and on the PATH ('make osio')
#
# Use:
# - Create file logins.txt with the test user credentials.
#   One accounts per line and the format <username><TAB><password>
# - Run './siegeJenkins.sh'
#

LOGINS=logins.txt
URLS=urls.txt
JENKINS_BASE_URL=https://jenkins.prod-preview.openshift.io
JENKINS_URLS=(${JENKINS_BASE_URL}/ ${JENKINS_BASE_URL}/view/all/builds/ ${JENKINS_BASE_URL}/computer/)

while getopts "c:d:t:" opt; do
  case ${opt} in
    c ) CONCURRENT=$OPTARG
      ;;
    d ) DELAY=$OPTARG
      ;;
    t ) TIME=$OPTARG
      ;;
  esac
done

CONCURRENT="${CONCURRENT:-25}"
DELAY="${DELAY:-10}"
TIME="${TIME:-5M}"

rm $URLS 2> /dev/null

if [ ! -f ${LOGINS} ]; then
    echo "${LOGINS} not found!"
fi

while read user    pass; do
  json_token=$(osio jwt -u ${user} -p ${pass} -e true)
  for url in "${JENKINS_URLS[@]}"
  do
	echo "${url}?token_json=${json_token}" >> ${URLS}
  done

done < ${LOGINS}

siege -R siegerc -c${CONCURRENT} -d${DELAY} -t${TIME} -f urls.txt
