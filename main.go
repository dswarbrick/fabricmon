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
// #cgo LDFLAGS: -libumad
// #include <stdlib.h>
// #include <umad.h>
import "C"

import (
	"fmt"
	"unsafe"
)

func main() {
	caNames, _ := getCANames()

	for _, caName := range caNames {
		var ca C.umad_ca_t

		fmt.Printf("umad_get_ca(\"%s\")\n", caName)

		// Pointer to char array will be allocated on C heap; must free pointer explicitly
		ca_name := C.CString(caName)

		// TODO: Replace umad_get_ca() with pure Go implementation
		if ret := C.umad_get_ca(ca_name, &ca); ret == 0 {
			fmt.Printf("Found CA %s (%s) with %d ports and firmware %s\n",
				C.GoString(&ca.ca_name[0]), C.GoString(&ca.ca_type[0]), ca.numports, C.GoString(&ca.fw_ver[0]))
		}

		C.free(unsafe.Pointer(ca_name))
	}
}
