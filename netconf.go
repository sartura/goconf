package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"encoding/xml"

	"github.com/Juniper/go-netconf/netconf"
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

func getModuleNames(ctx *C.struct_ly_ctx, xpath string) string {
	xmlns := ""
	var idx C.uint32_t
	idx = 0

	for {
		/* get all module names from libyang context */
		module := C.ly_ctx_get_module_iter((*C.const_ctx)(unsafe.Pointer(ctx)), &idx)
		if nil == module {
			break
		}
		/* if the module name is present in the xpath add it to the namespace */
		if strings.Contains(xpath, C.GoString(module.name)) {
			xmlns = xmlns + " xmlns:" + C.GoString(module.name) + "='" + C.GoString(module.ns) + "' "
		}
	}

	return xmlns
}

func netconfOperation(s *netconf.Session, ctx *C.struct_ly_ctx, datastore string, xpath string, value string, operationString string) error {

	cXpath := C.CString(xpath)
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cXpath))
	defer C.free(unsafe.Pointer(cValue))

	var operation int
	switch {
	case operationString == "get-config":
		operation = C.LYD_OPT_GETCONFIG
	case operationString == "get":
		operation = C.LYD_OPT_GET
	case operationString == "set":
		operation = C.LYD_OPT_EDIT
	}

	var xpathXML *C.char
	var xmlns = ""
	if operation == C.LYD_OPT_EDIT {
		node := C.lyd_new_path(nil, ctx, cXpath, unsafe.Pointer(cValue), 0, 0)
		if nil == node {
			return errors.New("libyang error: lyd_new_path")
		}
		defer C.lyd_free_withsiblings(node)

		C.lyd_print_mem(&xpathXML, node, C.LYD_XML, 0)
		if nil == xpathXML {
			return errors.New("libyang error: lyd_print_mem")
		}
		defer C.free(unsafe.Pointer(xpathXML))
	} else {
		xmlns = getModuleNames(ctx, xpath)
	}

	netconfXML := ""
	switch {
	case operation == C.LYD_OPT_GETCONFIG:
		netconfXML = "<get-config><source><" + datastore + "/></source><filter " + xmlns + " type=\"xpath\" select=\"" + xpath + "\"></filter></get-config>"
	case operation == C.LYD_OPT_GET:
		netconfXML = "<get><filter " + xmlns + " type=\"xpath\" select=\"" + xpath + "\"></filter></get>"
	case operation == C.LYD_OPT_EDIT:
		netconfXML = "<edit-config><target><" + datastore + "/></target><config>" + C.GoString(xpathXML) + "</config></edit-config>"
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
	cReplyData := C.CString(reply.Data)
	defer C.free(unsafe.Pointer(cReplyData))
	lyxml := C.lyxml_parse_mem(ctx, cReplyData, C.LYXML_PARSE_MULTIROOT)
	if lyxml == nil {
		return errors.New("libyang error: lyxml_parse_mem")
	}
	defer C.lyxml_free(ctx, lyxml)
	var xmlNoData *C.char
	C.lyxml_print_mem(&xmlNoData, lyxml.child, C.LYXML_PRINT_SIBLINGS)
	if xmlNoData == nil {
		return errors.New("libyang error: lyxml_print_mem")
	}
	defer C.free(unsafe.Pointer(xmlNoData))

	dataNode := C.go_lyd_parse_mem(ctx, xmlNoData, C.LYD_XML, C.int(operation))
	if dataNode == nil {
		return errors.New("libyang error: go_lyd_parse_mem")
	}
	defer C.lyd_free_withsiblings(dataNode)

	set := C.lyd_find_path(dataNode, cXpath)
	if set == nil {
		return errors.New("libyang error: lyd_find_path")
	}
	defer C.ly_set_free(set)

	if set.number == 0 {
		return errors.New("No data found: ly_set_free")
	}

	C.printSet(set)

	return nil
}

func getRemoteContext(s *netconf.Session) (*C.struct_ly_ctx, error) {
	var err error
	ctx := C.ly_ctx_new(nil, 0)

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
		return nil, errors.New("failed to fetch YANG schemas")
	}

	var data data
	err = xml.Unmarshal([]byte(reply.Data), &data)
	if err != nil {
		return nil, errors.New("Failed to parse YANG response")
	}

	getSchema := `
	<get-schema xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring"><identifier>%s</identifier><version>%s</version><format>%s</format></get-schema>
	`
	for i := range data.Schema {
		if data.Schema[i].Format == "yang" {
			schema := data.Schema[i]
			if "ietf-yang-library" == schema.Identifier {
				continue
			}
			request := fmt.Sprintf(getSchema, schema.Identifier, schema.Version, schema.Format)
			reply, err := s.Exec(netconf.RawMethod(request))
			if err != nil {
				fmt.Printf("init data ERROR: %s\n", err)
			}
			var yang string
			err = xml.Unmarshal([]byte(reply.Data), &yang)
			if err != nil {
				return nil, errors.New("Failed to parse YANG response")
			}
			cYang := C.CString(yang)
			defer C.free(unsafe.Pointer(cYang))
			module := C.lys_parse_mem(ctx, cYang, C.LYS_IN_YANG)
			if module == nil {
				C.ly_errmsg(ctx)
				return nil, errors.New("libyang error on lys_parse_mem")
			}
			for i := 0; i < int(module.features_size); i++ {
				rc := C.lys_features_enable(module, C.get_features(module.features, (C.int)(i)))
				if 0 != rc {
					fmt.Printf("Could not enable feature\n")
				}
			}
		}
	}

	//lrc = lys_features_enable(module, schemas[s].enabled_features[i]);
	//if (0 != lrc) {
	//ERR("The feature %s is not defined in the yang model %s", schemas[s].enabled_features[i], ctx->yang_model);

	/* send message to keep the connection alive */
	go func() {
		HelloMessage := ``

		ticker := time.NewTicker(18 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, err := s.Exec(netconf.RawMethod(HelloMessage))
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
	//C.ly_set_log_clb((C.clb)(unsafe.Pointer(C.CErrorCallback)), 0)

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
