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

func caDiscoverFabric(ca C.umad_ca_t, output chan Fabric) {
	hostname, _ := os.Hostname()
	caName := C.GoString(&ca.ca_name[0])

	mgmt_classes := [3]C.int{C.IB_SMI_CLASS, C.IB_SA_CLASS, C.IB_PERFORMANCE_CLASS}

	// Iterate over CA's umad_port array
	for _, umad_port := range ca.ports {
		// ca.ports may contain noncontiguous umad_port pointers
		if umad_port == nil {
			continue
		}

		portNum := int(umad_port.portnum)
		log.Printf("Polling %s port %d", caName, portNum)

		// ibnd_config_t specifies max hops, timeout, max SMPs etc
		var config C.ibnd_config_t

		// NOTE: Under ibsim, this will fail after a certain number of iterations with a
		// mad_rpc_open_port() errors (presumably due to a resource leak in ibsim).
		// ibnd_fabric_t *ibnd_discover_fabric(char *ca_name, int ca_port, ib_portid_t *from, ibnd_config_t *config)
		fabric, err := C.ibnd_discover_fabric(&ca.ca_name[0], umad_port.portnum, nil, &config)

		if err != nil {
			log.Println("Unable to discover fabric:", err)
			continue
		}

		// Open MAD port, which is needed for getting port counters.
		// struct ibmad_port *mad_rpc_open_port(char *dev_name, int dev_port, int *mgmt_classes, int num_classes)
		mad_port := C.mad_rpc_open_port(&ca.ca_name[0], umad_port.portnum, &mgmt_classes[0], C.int(len(mgmt_classes)))

		if mad_port != nil {
			nodes := walkFabric(fabric, mad_port)
			C.mad_rpc_close_port(mad_port)

			if output != nil {
				output <- Fabric{
					Hostname:   hostname,
					CAName:     caName,
					SourcePort: portNum,
					Nodes:      nodes,
				}
			}
		} else {
			log.Printf("ERROR: Unable to open MAD port: %s: %d", caName, portNum)
		}

		C.ibnd_destroy_fabric(fabric)
	}
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
