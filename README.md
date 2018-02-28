# libyang

[libyang](https://github.com/CESNET/libyang) is YANG data modeling language parser and toolkit written (and
providing API) in C. The library is used e.g. in [libnetconf2](https://github.com/CESNET/libnetconf2),
[Netopeer2](https://github.com/CESNET/Netopeer2) or [sysrepo](https://github.com/sysrepo/sysrepo) projects.

## Building libyang

You can simply install locally libyang with executing the following steps.

```
$ git clone https://github.com/CESNET/libyang.git
$ cd libyang
$ mkdir build; cd build
$ cmake ..
$ make
# make install
```

## Include libyang into your project

To use the libyang library in your go project add these lines of code under your import statement.

```
/*
#cgo LDFLAGS: -lyang
#include <libyang/libyang.h>
*/
import "C"

```

For a static build also add pcre or you will get errors.

```
#cgo LDFLAGS: -lpcre
```

## Compile your code

To create a binary that is dynamically linked to libyang simply run:

```
go build
```

For a static build execute "go build" with additional flags:
```
go build --ldflags '-extldflags "-static"'
```

## goconf command line

For help use:

```
./goconf --help
Usage of ./goconf:
  -datastore string
    	datastore used for get-config and edit operation (default "running")
  -edit string
    	-edit <xpath>, add/edit leaf and leaf-lists
  -get string
    	-get <xpath>
  -get-config string
    	-get-config <xpath>
  -ip string
    	ip address of the NETCONF server (default "localhost")
  -password string
    	NETCONF password (default "root")
  -port string
    	port of the NETCONF server (default "830")
  -username string
    	NETCONF username (default "root")
  -value string
    	-value <value>, value for edit operation
```
