#!/bin/bash

cd /opt/dev/go/src/github.com/sartura/gopath
go get
go build -a --ldflags '-extldflags " -ldl -static"'
