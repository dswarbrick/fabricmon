/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * cgo wrapper around libibumad / libibnetdiscover
 * Note: Due to the usual permissions on /dev/infiniband/umad*, this will probably need to be
 * executed as root.
 *
 * TODO: Implement user-friendly display of link / speed / rate etc. (see ib_types.h)
 */
package main

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libmad -libumad -libnetdisc
// #include <stdlib.h>
// #include <umad.h>
// #include <ibnetdisc.h>
import "C"

import (
	"fmt"
	"os"
	"sync"
	"unsafe"
)

type Fabric struct {
	mutex      sync.RWMutex
	ibndFabric *C.struct_ibnd_fabric
	ibmadPort  *C.struct_ibmad_port
}

// iterateSwitches walks the null-terminated node linked-list in f.nodes, displaying only swtich
// nodes
func iterateSwitches(f *Fabric, nnMap *NodeNameMap) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for node := f.ibndFabric.nodes; node != nil; node = node.next {
		if node._type == C.IB_NODE_SWITCH {
			var portid C.ib_portid_t

			fmt.Printf("Node type: %d, node descr: %s, num. ports: %d, node GUID: %#016x\n\n",
				node._type, nnMap.remapNodeName(uint64(node.guid), C.GoString(&node.nodedesc[0])),
				node.numports, node.guid)

			C.ib_portid_set(&portid, C.int(node.smalid), 0, 0)

			// node.ports is an array of pointers, in which any port may be null. We use pointer
			// arithmetic to get pointer to port struct.
			arrayPtr := uintptr(unsafe.Pointer(node.ports))

			for portNum := 0; portNum <= int(node.numports); portNum++ {
				// Get pointer to port struct and increment arrayPtr to next pointer
				pp := *(**C.ibnd_port_t)(unsafe.Pointer(arrayPtr))
				arrayPtr += unsafe.Sizeof(arrayPtr)

				portState := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_STATE_F)
				physState := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_PHYS_STATE_F)

				// TODO: Decode EXT_PORT_LINK_SPEED (i.e., FDR10)
				linkWidth := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_WIDTH_ACTIVE_F)
				linkSpeed := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_SPEED_ACTIVE_F)

				fmt.Printf("Port %d, port state: %d, phys state: %d, link width: %d, link speed: %d\n",
					portNum, portState, physState, linkWidth, linkSpeed)

				// TODO: Rework portState checking to optionally decode counters regardless of portState
				if portState != C.IB_LINK_DOWN {
					fmt.Printf("port %#v\n", pp)

					// This should not be nil if the link is up, but check anyway
					// FIXME: portState may be polling / armed etc, and rp will be null!
					rp := pp.remoteport
					if rp != nil {
						fmt.Printf("Remote node type: %d, GUID: %#016x, descr: %s\n",
							rp.node._type, rp.node.guid,
							nnMap.remapNodeName(uint64(node.guid), C.GoString(&rp.node.nodedesc[0])))
					}
				}
			}
		}
	}
}

func main() {
	caNames, _ := getCANames()

	nnMap, _ := NewNodeNameMap()

	for _, caName := range caNames {
		var ca C.umad_ca_t

		// Pointer to char array will be allocated on C heap; must free pointer explicitly
		ca_name := C.CString(caName)

		// TODO: Replace umad_get_ca() with pure Go implementation
		if ret := C.umad_get_ca(ca_name, &ca); ret == 0 {
			var (
				config C.ibnd_config_t
				err    error
				fabric Fabric
			)

			fmt.Printf("Found CA %s (%s) with %d ports and firmware %s\n",
				C.GoString(&ca.ca_name[0]), C.GoString(&ca.ca_type[0]), ca.numports, C.GoString(&ca.fw_ver[0]))
			fmt.Printf("Node GUID: %#016x, system GUID: %#016x\n\n",
				ntohll(uint64(ca.node_guid)), ntohll(uint64(ca.system_guid)))

			fmt.Printf("%s: %#v\n\n", caName, ca)

			for p := 1; ca.ports[p] != nil; p++ {
				fmt.Printf("port %d: %#v\n\n", p, ca.ports[p])
			}

			// Return pointer to fabric struct
			fabric.ibndFabric, err = C.ibnd_discover_fabric(&ca.ca_name[0], 1, nil, &config)

			if err != nil {
				fmt.Println("Unable to discover fabric:", err)
				os.Exit(1)
			}

			mgmt_classes := [3]C.int{C.IB_SMI_CLASS, C.IB_SA_CLASS, C.IB_PERFORMANCE_CLASS}
			fabric.ibmadPort, err = C.mad_rpc_open_port(ca_name, 1, &mgmt_classes[0], C.int(len(mgmt_classes)))

			if err != nil {
				fmt.Println("Unable to open MAD port:", err)
				os.Exit(1)
			}

			fmt.Printf("ibmad_port: %#v\n", fabric.ibmadPort)

			// Walk switch nodes in fabric
			iterateSwitches(&fabric, &nnMap)

			fabric.mutex.Lock()

			// Close MAD port
			C.mad_rpc_close_port(fabric.ibmadPort)

			// Free memory and resources associated with fabric
			C.ibnd_destroy_fabric(fabric.ibndFabric)

			fabric.mutex.Unlock()
		}

		C.free(unsafe.Pointer(ca_name))
	}
}
