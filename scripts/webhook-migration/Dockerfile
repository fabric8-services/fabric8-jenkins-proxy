FROM centos:7
LABEL maintainer="hferents@redhat.com"

ENV LANG=en_US.utf8
ENV OS_CLI_VERSION v3.7.0
ENV OS_TAG 7ed6862

RUN yum install -y epel-release
RUN yum --enablerepo=centosplus install -y \
      net-tools \
      git \
      golang \
      make \
      jq \
      curl \
      which \
    && yum clean all

# Get oc installed
# Don't care about minimal numbers of layers
RUN curl -s -L https://github.com/openshift/origin/releases/download/${OS_CLI_VERSION}/openshift-origin-client-tools-${OS_CLI_VERSION}-${OS_TAG}-linux-64bit.tar.gz -o /tmp/oc.tar.gz
RUN tar zxvf /tmp/oc.tar.gz -C /tmp/
RUN mv /tmp/openshift-origin-client-tools-${OS_CLI_VERSION}-${OS_TAG}-linux-64bit/oc /usr/local/bin/
RUN rm -rf /tmp/oc.tar.gz /tmp/openshift-origin-client-tools-${OS_CLI_VERSION}-${OS_TAG}-linux-64bit/
RUN rm -rf /root/.cache /var/cache/apk/
RUN oc version

# Setup Go
ENV GOROOT /usr/lib/golang
ENV PATH=$GOROOT/bin:$PATH

ENV GOPATH=$HOME/go
RUN mkdir -p $GOPATH/src
RUN mkdir -p $GOPATH/pkg
RUN mkdir -p $GOPATH/bin

ENV GOBIN=$GOPATH/bin
ENV PATH=$GOBIN:$PATH
RUN curl https://glide.sh/get | sh

# Get osio installed
ENV ORG=github.com/fabric8-services
ENV REPO=$ORG/fabric8-jenkins-proxy
ENV BUILD_DIR=$GOPATH/src/$REPO
RUN mkdir -p $GOPATH/src/$ORG
RUN git clone https://$REPO $BUILD_DIR
RUN cd $BUILD_DIR && git rev-parse --short HEAD && make osio

COPY updateWebhooks.sh /usr/local/bin/updateWebhooks.sh

ENTRYPOINT [ "updateWebhooks.sh" ]
