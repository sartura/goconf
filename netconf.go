package main

import (
	"errors"
	"fmt"
	"unsafe"

	"encoding/xml"

	"github.com/sartura/go-netconf/netconf"
)

/*
#cgo LDFLAGS: -lyang
#cgo LDFLAGS: -lpcre
#include <libyang/libyang.h>
#include <libyang/tree_data.h>

#include <stdlib.h>
#include "helper.h"
*/
import "C"

type data struct {
	XMLName xml.Name `xml:"data"`
	Schema  []schema `xml:"netconf-state>schemas>schema"`
}

type schema struct {
	XMLName    xml.Name `xml:"schema"`
	Identifier string   `xml:"identifier"`
	Version    string   `xml:"version"`
	Format     string   `xml:"format"`
	Namespace  string   `xml:"namespace"`
	Location   string   `xml:"location"`
}

func netconfOperation(s *netconf.Session, ctx *C.struct_ly_ctx, xpath string, value *string, operation string) (string, error) {

	var libyangVal *C.char
	libyangVal = nil
	if value != nil {
		libyangVal = C.CString((*value))
	}

	node := C.lyd_new_path(nil, ctx, C.CString(xpath), unsafe.Pointer(libyangVal), C.LYD_ANYDATA_XML, 0)
	defer C.lyd_free_withsiblings(node)

	var str *C.char
	C.lyd_print_mem(&str, node, C.LYD_XML, 0)
	defer C.free(unsafe.Pointer(str))

	xml := ""
	switch {
	case operation == "get-config":
		xml = "<get-config><source><running/></source><filter>" + C.GoString(str) + "</filter></get-config>"
	case operation == "get":
		xml = "<get><filter type=\"subtree\">" + C.GoString(str) + "</filter></get>"
	case operation == "set":
		xml = "<edit-config xmlns:nc='urn:ietf:params:xml:ns:netconf:base:1.0'><target><running/></target><config>" + C.GoString(str) + "</config></edit-config>"
	default:
		return "", errors.New("invalid operation")
	}

	reply, err := s.Exec(netconf.RawMethod(xml))
	if err != nil {
		println("ERROR: ", err)
		return "", err
	}

	return reply.Data, nil
}

func getRemoteContext(s *netconf.Session) (*C.struct_ly_ctx, error) {
	var err error
	ctx := C.ly_ctx_new(nil)

	getSchemas := `
	<get>
	<filter type="subtree">
	<netconf-state xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring">
	<schemas/>
	</netconf-state>
	</filter>
	</get>
	`
	// Sends raw XML
	reply, err := s.Exec(netconf.RawMethod(getSchemas))
	if err != nil {
		return nil, errors.New("libyang parse error")
	}

	var data data
	err = xml.Unmarshal([]byte(reply.Data), &data)
	if err != nil {
		return nil, errors.New("libyang parse error")
	}

	getSchema := `
	<get-schema xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring"><identifier>%s</identifier><version>%s</version><format>%s</format></get-schema>
	`
	for i := range data.Schema {
		if data.Schema[i].Format == "yang" {
			schema := data.Schema[i]
			request := fmt.Sprintf(getSchema, schema.Identifier, schema.Version, schema.Format)
			reply, err := s.Exec(netconf.RawMethod(request))
			if err != nil {
				fmt.Printf("init data ERROR: %s\n", err)
			}
			var yang string
			err = xml.Unmarshal([]byte(reply.Data), &yang)
			if err != nil {
				return nil, errors.New("libyang parse error")
			}
			module := C.lys_parse_mem(ctx, C.CString(yang), C.LYS_IN_YANG)
			if module == nil {
				return nil, errors.New("libyang parse error")
			}
		}
	}

	return ctx, nil
}

//export GoErrorCallback
func GoErrorCallback(level C.LY_LOG_LEVEL, msg *C.char, path *C.char) {
	//log.Printf("libyang error: %s\n", C.GoString(msg))
	return
}

func getNetconfContext() (*C.struct_ly_ctx, *netconf.Session) {
	// prepare libyang logs
	C.ly_set_log_clb((C.clb)(unsafe.Pointer(C.CErrorCallback)), 0)

	var ctx *C.struct_ly_ctx
	ctx = nil

	var s *netconf.Session
	s = nil

	return ctx, s
}

func cleanNetconfContext(ctx *C.struct_ly_ctx, s *netconf.Session) {
	if ctx != nil {
		C.ly_ctx_destroy(ctx, nil)
	}

	if s != nil {
		s.Close()
	}
}
