#!/bin/bash
#
# Used to update GitHub webhooks.
# See https://github.com/fabric8-services/fabric8-jenkins-proxy/issues/94

# Make sure we have all the required environment variables injected
[ -z "${TENANT_URL}" ] && echo "TENANT_URL needs to be set." && exit 1
[ -z "${AUTH_URL}" ] && echo "AUTH_URL needs to be set" && exit 1
[ -z "${OPENSHIFT_URL}" ] && echo "OPENSHIFT_URL needs to be set." && exit 1
[ -z "${OPENSHIFT_API_TOKEN}" ] && echo "OPENSHIFT_API_TOKEN needs to be set." && exit 1
[ -z "${AUTH_TOKEN}" ] && echo "AUTH_TOKEN needs to be set." && exit 1
[ -z "${JENKINS_OLD_URL_SUFFIX}" ] && echo "JENKINS_OLD_URL_SUFFIX needs to be set." && exit 1
[ -z "${JENKINS_PROXY_URL}" ] && echo "JENKINS_PROXY_URL needs to be set." && exit 1
[ -z "${PRIVATE_KEY}" ] && echo "PRIVATE_KEY needs to be set." && exit 1
[ -z "${TARGET_ENV}" ] && echo "TARGET_ENV needs to be set. [prod|stage]." && exit 1

DRY_RUN="${DRY_RUN:-false}"

# Ensure required tools are installed
hash jq 2>/dev/null || { echo >&2 "'jq' is required."; exit 1; }
hash curl 2>/dev/null || { echo >&2 "'curl' is required."; exit 1; }
hash oc 2>/dev/null || { echo >&2 "'oc' is required."; exit 1; }
hash osio 2>/dev/null || { echo >&2 "'osio' is required."; exit 1; }

###############################################################################
# Processes the build configuration of
# the OpenShift cluster and extracts GitHub
# repo URLs which might need updating.
# Globals:
#   OPENSHIFT_URL       - The OpenShift URL containing the user namespaces
#   OPENSHIFT_API_TOKEN - API token to access OpenShift. Needs to have permission to list all namespaces
#   CHECK_NAMESPACES    - List of namespaces to check. For testing purposes. If not set all namespaces are iterated
#   NAME_SPACES         - Array of namespaces which contain a git URL one of their build configurations
#                         This array is populated as part of the execution of this method.
#   REPOS               - Array of Git repository URLs aligning by index with NAME_SPACES.
#                       - This array is populated as part of the execution of this method.
# Arguments:
#   None
# Returns:
#   None
###############################################################################
findGitHubRepos() {
  export KUBECONFIG=/tmp/config
  echo "Using KUBECONFIG ${KUBECONFIG}"
  oc login ${OPENSHIFT_URL} --token=${OPENSHIFT_API_TOKEN} > /dev/null

  if [ -z "${CHECK_NAMESPACES}" ] ; then
    CHECK_NAMESPACES=$(oc projects -q)
  fi
  for namespace in ${CHECK_NAMESPACES}; do
   printf "Processing namespace ${namespace}"
   local tmp=($(oc get bc -n ${namespace} -o json | jq -r ' .items[] | select(.spec.source.git.uri) | .metadata.namespace + " " + .spec.source.git.uri[19:-4]'))
   if [ ${#tmp[@]} -eq 0  ]; then
    printf "\n"
    continue
   fi
   printf " *\n"
   NAME_SPACES+=(${tmp[0]})
   REPOS+=(${tmp[1]})
  done
}

###############################################################################
# Finds the UUIDs for the discovered namespaces
# Globals:
#   NAME_SPACES - Namespaces to iterate
#   UUIDS       - Array which gets populated with UUIDs
# Arguments:
#   None
# Returns:
#   None
###############################################################################
findUUIDs() {
  for i in "${!NAME_SPACES[@]}"; do
    echo "Determining user UUId for namespace ${namespace}"
    uuid=$(curl -sgSL -H "Authorization: Bearer ${AUTH_TOKEN}" ${TENANT_URL}api/tenants?namespace=${NAME_SPACES[$i]}\&master_url=${OPENSHIFT_URL} | jq -r .data[0].id)
     if [ ${#uuid} -le 5 ]; then
        echo "WARN: Unable to determine UUID for ${NAME_SPACES[$i]}." >&2
        echo "WARN: Request URL ${TENANT_URL}api/tenants?namespace=${NAME_SPACES[$i]}\&master_url=${OPENSHIFT_URL}" >&2
      else
        UUIDS[$i]=${uuid}
      fi
  done
}

###############################################################################
# For each UUID get an OSIO token
# Globals:
#   UUIDS      - User ids to iterate
#   OSIO_TOKEN - Array which gets populated with OSIO tokens
# Arguments:
#   None
# Returns:
#   None
###############################################################################
getOsioTokens() {
  for i in "${!UUIDS[@]}"; do
    echo "Generating OSIO token for UUID ${UUIDS[$i]}"
    osio_token=$(osio -t ${TARGET_ENV} token -k "${PRIVATE_KEY}" -u ${UUIDS[$i]})
    OSIO_TOKENS[$i]="${osio_token}"
  done
}

###############################################################################
# For each OSIO token get the GitHub token
# Globals:
#   OSIO_TOKENS    - OSIO tokens to iterate
#   GITHUB_TOKENS  - Array which gets populated with GitHub tokens
#   UUIDS          - User ids (for logging)
# Arguments:
#   None
# Returns:
#   None
###############################################################################
getGitHubTokens() {
  for i in "${!OSIO_TOKENS[@]}"; do
    [ -z "${OSIO_TOKENS[${i}]}"  ] && echo "WARN: Empty OSIO token for namespace ${NAME_SPACES[$i]}" >&2 && continue

    gh_token=$(curl -sgSL -H "Authorization: Bearer ${OSIO_TOKENS[${i}]}" ${AUTH_URL}api/token?for=https://github.com | jq -r '.access_token')
    if [ "${gh_token}" == "null" ]; then
        echo "WARN: Unable to create GitHub token for UUID ${UUIDS[$i]}" >&2
        curl -sgSL -H "Authorization: Bearer ${OSIO_TOKENS[${i}]}" ${AUTH_URL}api/token?for=https://github.com >&2
    else
        GITHUB_TOKENS[$i]=${gh_token}
    fi
  done
}

###############################################################################
# For each OSIO token get the GitHub token
# See https://developer.github.com/v3/repos/hooks
# Globals:
#   REPOS          - List of repositories to update
#   GITHUB_TOKENS  - For each repository its GitHub access token
#   NAME_SPACES    - For each repository its namespace (for logging)
# Arguments:
#   None
# Returns:
#   None
###############################################################################
updateGitHubWebhooks() {
  for i in "${!REPOS[@]}"; do
    [ -z "${GITHUB_TOKENS[${i}]}"  ] && echo "WARN: Empty GitHub token for namespace ${NAME_SPACES[$i]}" >&2 && continue

    while read id url; do
        if [[ ${url} =~ .*${JENKINS_OLD_URL_SUFFIX}.*$ ]]; then
          if [ "${DRY_RUN}" = true ] ; then
            echo DRY_RUN for ${url} update:
            echo curl -sgSL -H "Authorization: token ***" -X PATCH \
            --data '{"config": {"url": "'${JENKINS_PROXY_URL}'github-webhook/"}}' https://api.github.com/repos/${REPOS[$i]}/hooks/${id}
            echo " "
          else
            response=$(curl -sgSL -H "Authorization: token ${GITHUB_TOKENS[$i]}" -X PATCH \
              --data '{"config": {"url": "'${JENKINS_PROXY_URL}'github-webhook/"}}' \
              https://api.github.com/repos/${REPOS[$i]}/hooks/${id} | \
              jq -r '.last_response.message')
            echo "${url} updated. Status: $response"
          fi
        else
          echo "Skipping GitHub webhook '${url}' (no match)"
        fi
    done < <(curl -sgSL -H "Authorization: token ${GITHUB_TOKENS[$i]}" \
      https://api.github.com/repos/${REPOS[$i]}/hooks |  jq -r '.[] |  if .id? != null then (.id|tostring) + " " + .config.url else "- -"  end')
  done
}

###############################################################################
# Checks whether the specified string ends with a '/' and if not adds one.
# Globals:
#   None
# Arguments:
#   $1       - The string to check for the trailing slash.
# Returns:
#   The unmodified string as passed to the function in case the string already has
#   a trailing slash, otherwise the passed string with a '/' appended.
###############################################################################
ensureTrailingSlash() {
    length=${#1}
    last_char=${1:length-1:1}

    if [ "${last_char}" == "/" ] ; then
        echo "$1"
    else
        echo "$1/"
    fi
}

# Make sure all URLs are ending consistently in a '/'
OPENSHIFT_URL=$(ensureTrailingSlash ${OPENSHIFT_URL})
JENKINS_PROXY_URL=$(ensureTrailingSlash ${JENKINS_PROXY_URL})
AUTH_URL=$(ensureTrailingSlash ${AUTH_URL})
TENANT_URL=$(ensureTrailingSlash ${TENANT_URL})

# Declare a bunch of parallel arrays which we collect the required data in
declare -a NAME_SPACES
declare -a REPOS
declare -a UUIDS
declare -a OSIO_TOKENS
declare -a GITHUB_TOKENS

# Do all the prep work
echo "Determining affected namespaces and repositories "
findGitHubRepos
echo "Finding UUIDs of affected users"
findUUIDs
echo "Generating temporary OSIO tokens"
getOsioTokens
echo "Determining GitHub tokens for webhook update"
getGitHubTokens

# Dump info on what we collected
echo " "
for i in "${!NAME_SPACES[@]}"; do
  echo "ns         : ${NAME_SPACES[$i]}"
  echo "uuid       : ${UUIDS[$i]}"
  echo "repo       : ${REPOS[$i]}"
  if [ -z "${OSIO_TOKENS[$i]}"  ]; then
    echo "osio-token : WARN: Not set"
  else
    echo "osio-token : ***"
  fi
  if [ -z "${GITHUB_TOKENS[$i]}"  ]; then
    echo "gh-token   : WARN: Not set"
  else
    echo "gh-token   : ***"
  fi
  echo " "
done

# Let's do the update
updateGitHubWebhooks
