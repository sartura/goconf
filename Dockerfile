FROM sysrepo/sysrepo-netopeer2:latest

MAINTAINER mislav.novakovic@sartura.hr

RUN \
      apt-get update && apt-get install -y \
      golang

# use /opt/dev as working directory
WORKDIR /opt/dev

# goconf
RUN \
      git clone https://github.com/sartura/goconf.git && \
      cd goconf && \
      go get github.com/Juniper/go-netconf/netconf && \
      go get github.com/chzyer/readline && \
      go build
