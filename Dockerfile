FROM ubuntu:14.04
MAINTAINER Nitrous.IO <eng@nitrous.io>

RUN apt-get update && apt-get install -y \
  build-essential \
  curl \
  git-core \
  mercurial

RUN curl -s https://storage.googleapis.com/golang/go1.3.linux-amd64.tar.gz \
  | tar -C /usr/local -xz

RUN mkdir -p /opt/src/github.com/nitrous-io/ferry

ENV GOPATH /opt
ENV PATH /opt/bin:/usr/local/go/bin:$PATH
ENV WORKDIR /opt/src/github.com/nitrous-io/ferry

ADD util/ssh_config /etc/ssh/ssh_config
ADD . /opt/src/github.com/nitrous-io/ferry

WORKDIR /opt/src/github.com/nitrous-io/ferry
