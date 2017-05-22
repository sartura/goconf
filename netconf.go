package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
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

var showLibyangLogs bool

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

func printRecursiveXPATH(node *C.struct_lyd_node, first bool, parent *C.struct_lyd_node) {

	if node == nil {
		return
	}

	if parent != nil {
		if node.parent != parent {
			return
		}
	}

	if node.schema.nodetype == C.LYS_LEAF || node.schema.nodetype == C.LYS_LEAFLIST {
		stringXpath := C.lyd_path(node)
		println(C.GoString(stringXpath) + " " + C.GoString((*C.struct_lyd_node_leaf_list)(unsafe.Pointer(node)).value_str))
		C.free(unsafe.Pointer(stringXpath))
		if first {
			return
		}
	}

	if node.next != nil {
		printRecursiveXPATH(node.next, false, node.parent)
	}

	if node.child != nil {
		printRecursiveXPATH(node.child, false, node)
	}

	return
}

func getLastNonLeafNode(ctx *C.struct_ly_ctx, xpath string) *C.struct_lyd_node {
	var node *C.struct_lyd_node
	var tmp *C.struct_lyd_node

	items := strings.Split(xpath, "/")
	newXpath := ""

	//turn off libyang logs
	tmpShowLibyangLogs := showLibyangLogs
	for item := range items {
		newXpath = newXpath + items[item]
		tmp = C.lyd_new_path(nil, ctx, C.CString(newXpath), nil, 0, 0)
		if tmp != nil {
			C.lyd_free_withsiblings(node)
			node = tmp
		}
		newXpath = newXpath + "/"
	}
	showLibyangLogs = tmpShowLibyangLogs

	return node
}

func netconfOperation(s *netconf.Session, ctx *C.struct_ly_ctx, datastore string, xpath string, value string, operationString string) error {

	var operation int
	switch {
	case operationString == "get-config":
		operation = C.LYD_OPT_GETCONFIG
	case operationString == "get":
		operation = C.LYD_OPT_GET
	case operationString == "set":
		operation = C.LYD_OPT_EDIT
	}

	var node *C.struct_lyd_node
	if operation == C.LYD_OPT_EDIT {
		node = C.lyd_new_path(nil, ctx, C.CString(xpath), unsafe.Pointer(C.CString(value)), 0, 0)
	} else {
		node = getLastNonLeafNode(ctx, xpath)
	}
	if node == nil {
		return errors.New("libyang error: lyd_new_path")
	}
	defer C.lyd_free_withsiblings(node)

	var xpathXML *C.char
	C.lyd_print_mem(&xpathXML, node, C.LYD_XML, 0)
	if xpathXML == nil {
		return errors.New("libyang error: lyd_print_mem")
	}
	defer C.free(unsafe.Pointer(xpathXML))

	netconfXML := ""
	switch {
	case operation == C.LYD_OPT_GETCONFIG:
		netconfXML = "<get-config><source><" + datastore + "/></source><filter>" + C.GoString(xpathXML) + "</filter></get-config>"
	case operation == C.LYD_OPT_GET:
		netconfXML = "<get><filter type=\"subtree\">" + C.GoString(xpathXML) + "</filter></get>"
	case operation == C.LYD_OPT_EDIT:
		netconfXML = "<edit-config xmlns:nc='urn:ietf:params:xml:ns:netconf:base:1.0'><target><" + datastore + "/></target><config>" + C.GoString(xpathXML) + "</config></edit-config>"
	default:
		return errors.New("NETCONF: invalid operation")
	}

	reply, err := s.Exec(netconf.RawMethod(netconfXML))
	if err != nil {
		return err
	}

	if operation == C.LYD_OPT_EDIT {
		return nil
	}

	// remove top <data> xml node
	lyxml := C.lyxml_parse_mem(ctx, C.CString(reply.Data), C.LYXML_PARSE_MULTIROOT)
	if lyxml == nil {
		return errors.New("libyang error: lyxml_parse_mem")
	}
	defer C.lyxml_free(ctx, lyxml)
	var xmlNoData *C.char
	C.lyxml_print_mem(&xmlNoData, lyxml.child, 0)
	if xmlNoData == nil {
		return errors.New("libyang error: lyxml_print_mem")
	}
	defer C.free(unsafe.Pointer(xmlNoData))

	dataNode := C.go_lyd_parse_mem(ctx, xmlNoData, C.LYD_XML, C.int(operation))
	if dataNode == nil {
		return errors.New("libyang error: go_lyd_parse_mem")
	}
	defer C.lyd_free_withsiblings(dataNode)

	set := C.lyd_find_xpath(dataNode, C.CString(xpath))
	if set == nil {
		return errors.New("libyang error: lyd_find_xpath")
	}
	defer C.ly_set_free(set)

	if set.number == 0 {
		return errors.New("No data found: ly_set_free")
	}

	// bugfix, structs are wrongly allocated
	// test validity with parent comparison
	printRecursiveXPATH(C.get_item(set, C.int(0)), true, nil)

	return nil
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

	// hack to keep alive the connection
	//TODO fix this
	go func() {
		ticker := time.NewTicker(18 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, err := s.Exec(netconf.RawMethod("<keep-alive/>"))
			if err != nil {
				if err.Error() == "WaitForFunc failed" {
					return
				}
			}
		}
	}()

	return ctx, nil
}

//export GoErrorCallback
func GoErrorCallback(level C.LY_LOG_LEVEL, msg *C.char, path *C.char) {
	if showLibyangLogs {
		log.Printf("libyang error: %s\n", C.GoString(msg))
	}
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
