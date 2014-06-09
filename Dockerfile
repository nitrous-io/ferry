FROM ubuntu:14.04
MAINTAINER Nitrous.IO <eng@nitrous.io>

RUN apt-get update && apt-get install -y \
  build-essential \
  curl \
  git \
  mercurial \
  ruby1.9.1 \
  ruby1.9.1-dev

RUN curl -s https://storage.googleapis.com/golang/go1.2.2.linux-amd64.tar.gz \
  | tar -C /usr/local -xz

RUN gem install fpm --no-rdoc --no-ri --version 1.0.2

RUN mkdir -p /opt/src/github.com/nitrous-io/ferry

ENV VENDOR_PATH /opt/src/github.com/nitrous-io/ferry/_vendor
ENV GOPATH $VENDOR_PATH:/opt
ENV GOBIN $VENDOR_PATH/bin
ENV PATH $GOBIN:/usr/local/go/bin:$PATH
ENV WORKDIR /opt/src/github.com/nitrous-io/ferry

WORKDIR /opt/src/github.com/nitrous-io/ferry
