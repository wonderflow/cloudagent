FROM ubuntu:14.04

MAINTAINER Sun Jianbo <wonderflow.sun@gmail.com> (@wonderflow)

# Packaged dependencies
RUN apt-get update && apt-get install -y \
	automake \
	build-essential \
	curl \
	dpkg-sig \
	git \
	iptables \
	--no-install-recommends

# Install Go
ENV GO_VERSION 1.4.2
RUN curl -ksSL https://github.com/golang/go/archive/go${GO_VERSION}.tar.gz | tar -v -C /usr/local -xz \
	&& mkdir -p /go/bin
RUN mv /usr/local/go-go${GO_VERSION} /usr/local/go
ENV PATH /go/bin:/usr/local/go/bin:$PATH
RUN cd /usr/local/go/src && ./make.bash --no-clean 2>&1
ENV GOPATH /go:/go/src/github.com/wonderflow/cloudagent/Godeps/_workspace
RUN mkdir -p /var/vcap/monit

COPY . /go/src/github.com/wonderflow/cloudagent
WORKDIR /go/src/github.com/wonderflow/cloudagent
RUN cp /go/src/github.com/wonderflow/cloudagent/test/monit/monit.user /var/vcap/monit/


ENTRYPOINT ["build/build"]