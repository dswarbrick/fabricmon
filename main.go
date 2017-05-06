/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * cgo wrapper around libibumad / libibnetdiscover
 * Note: Due to the usual permissions on /dev/infiniband/umad*, this will probably need to be
 * executed as root.
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
	"unsafe"
)

type Fabric struct {
	ibndFabric *C.struct_ibnd_fabric
	ibmadPort  *C.struct_ibmad_port
}

func main() {
	caNames, _ := getCANames()

	for _, caName := range caNames {
		var ca C.umad_ca_t

		fmt.Printf("umad_get_ca(\"%s\")\n", caName)

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

			// Close MAD port
			C.mad_rpc_close_port(fabric.ibmadPort)

			// Free memory and resources associated with fabric
			C.ibnd_destroy_fabric(fabric.ibndFabric)
		}

		C.free(unsafe.Pointer(ca_name))
	}
}
