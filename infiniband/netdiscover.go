// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libnetdisc
// #include <umad.h>
// #include <ibnetdisc.h>
import "C"

import (
	"log"
	"os"
	"unsafe"
)

type HCA struct {
	Name        string

	// umad_ca_t contains an array of pointers - associated memory must be freed with
	// umad_release_ca(umad_ca_t *ca)
	umad_ca *C.umad_ca_t
}

func (h *HCA) NetDiscover(output chan Fabric) {
	hostname, _ := os.Hostname()

	mgmt_classes := [3]C.int{C.IB_SMI_CLASS, C.IB_SA_CLASS, C.IB_PERFORMANCE_CLASS}

	// Iterate over CA's umad_port array
	for _, umad_port := range h.umad_ca.ports {
		// ca.ports may contain noncontiguous umad_port pointers
		if umad_port == nil {
			continue
		}

		portNum := int(umad_port.portnum)
		log.Printf("Polling %s port %d", h.Name, portNum)

		// ibnd_config_t specifies max hops, timeout, max SMPs etc
		var config C.ibnd_config_t

		// NOTE: Under ibsim, this will fail after a certain number of iterations with a
		// mad_rpc_open_port() errors (presumably due to a resource leak in ibsim).
		// ibnd_fabric_t *ibnd_discover_fabric(char *ca_name, int ca_port, ib_portid_t *from, ibnd_config_t *config)
		fabric, err := C.ibnd_discover_fabric(&h.umad_ca.ca_name[0], umad_port.portnum, nil, &config)

		if err != nil {
			log.Println("Unable to discover fabric:", err)
			continue
		}

		// Open MAD port, which is needed for getting port counters.
		// struct ibmad_port *mad_rpc_open_port(char *dev_name, int dev_port, int *mgmt_classes, int num_classes)
		mad_port := C.mad_rpc_open_port(&h.umad_ca.ca_name[0], umad_port.portnum, &mgmt_classes[0], C.int(len(mgmt_classes)))

		if mad_port != nil {
			nodes := walkFabric(fabric, mad_port)
			C.mad_rpc_close_port(mad_port)

			if output != nil {
				output <- Fabric{
					Hostname:   hostname,
					CAName:     h.Name,
					SourcePort: portNum,
					Nodes:      nodes,
				}
			}
		} else {
			log.Printf("ERROR: Unable to open MAD port: %s: %d", h.Name, portNum)
		}

		C.ibnd_destroy_fabric(fabric)
	}
}

func (h *HCA) Release() {
	// Free associated memory from pointers in umad_ca_t.ports
	if C.umad_release_ca(h.umad_ca) < 0 {
		log.Printf("ERROR: umad_release_ca %#v\n", h.umad_ca)
	}
}

func GetCAs() []HCA {
	caNames := umadGetCANames()
	hcas := make([]HCA, len(caNames))

	for i, caName := range caNames {
		var ca C.umad_ca_t

		ca_name := C.CString(caName)
		C.umad_get_ca(ca_name, &ca)
		C.free(unsafe.Pointer(ca_name))

		log.Printf("Found CA %s (%s) with %d ports, firmware version: %s, hardware version: %s, "+
			"node GUID: %#016x, system GUID: %#016x\n",
			C.GoString(&ca.ca_name[0]), C.GoString(&ca.ca_type[0]), ca.numports,
			C.GoString(&ca.fw_ver[0]), C.GoString(&ca.hw_ver[0]),
			ntohll(uint64(ca.node_guid)), ntohll(uint64(ca.system_guid)))

		hcas[i] = HCA{
			Name:        caName,
			umad_ca:     &ca,
		}
	}

	return hcas
}

func walkFabric(fabric *C.struct_ibnd_fabric, mad_port *C.struct_ibmad_port) []Node {
	nodes := make([]Node, 0)

	for node := fabric.nodes; node != nil; node = node.next {
		myNode := Node{
			GUID:     uint64(node.guid),
			NodeType: int(node._type),
			NodeDesc: C.GoString(&node.nodedesc[0]),
			VendorID: uint16(C.mad_get_field(unsafe.Pointer(&node.info), 0, C.IB_NODE_VENDORID_F)),
			DeviceID: uint16(C.mad_get_field(unsafe.Pointer(&node.info), 0, C.IB_NODE_DEVID_F)),
		}

		log.Printf("node: %#v\n", myNode)

		if node._type == C.IB_NODE_SWITCH {
			myNode.Ports = walkPorts(node, mad_port)
		}

		nodes = append(nodes, myNode)
	}

	return nodes
}
