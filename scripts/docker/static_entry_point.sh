#!/bin/bash

cd /opt/dev/go/src/github.com/sartura/goconf
go get
go build -a --ldflags '-extldflags " -ldl -static"'
