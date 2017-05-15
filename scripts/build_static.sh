#!/bin/bash

# build docker
cd ./docker
docker build -t sysrepo/sysrepo-netopeer2:golang -f Dockerfile .
cd -

# compile golang code
pwd_dir=$(pwd)
APP_PATH=/opt/dev/go/src/github.com/sartura
docker run -i -t -v $pwd_dir/../../gopath:$APP_PATH/gopath --rm sysrepo/sysrepo-netopeer2:golang bash -c $APP_PATH/gopath/scripts/docker/static_entry_point.sh
