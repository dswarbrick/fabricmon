// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Subnet manager functions.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libmad -libumad
// #include <mad.h>
// #include <umad.h>
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	SMINFO_NOTACT uint8 = iota
	SMINFO_DISCOVER
	SMINFO_STANDBY
	SMINFO_MASTER
)

var smStateMap = [...]string{
	"SMINFO_NOTACT",
	"SMINFO_DISCOVER",
	"SMINFO_STANDBY",
	"SMINFO_MASTER",
}

// smInfo is a proof of concept function to get the SM info for a CA & port.
func smInfo(caName string, portNum int) {
	var (
		sminfo [1024]C.uint8_t
		guid   uint64
		act    uint16
	)

	mgmt_classes := [3]C.int{C.IB_SMI_CLASS, C.IB_SMI_DIRECT_CLASS, C.IB_SA_CLASS}

	ibd_ca := C.CString(caName)
	defer C.free(unsafe.Pointer(ibd_ca))

	ibd_ca_port := C.int(portNum)

	// struct ibmad_port *mad_rpc_open_port(char *dev_name, int dev_port, int *mgmt_classes, int num_classes)
	srcport := C.mad_rpc_open_port(ibd_ca, ibd_ca_port, &mgmt_classes[0], 3)

	prio := SMINFO_STANDBY
	state := SMINFO_STANDBY

	var portid C.ib_portid_t

	//C.resolve_sm_portid(ibd_ca, ibd_ca_port, &portid)
	var port C.umad_port_t

	C.umad_get_port(ibd_ca, ibd_ca_port, &port)
	portid.lid = C.int(port.sm_lid)
	portid.sl = C.uchar(port.sm_sl)
	C.umad_release_port(&port)

	C.mad_encode_field(&sminfo[0], C.IB_SMINFO_PRIO_F, unsafe.Pointer(&prio))
	C.mad_encode_field(&sminfo[0], C.IB_SMINFO_STATE_F, unsafe.Pointer(&state))

	C.smp_query_via(unsafe.Pointer(&sminfo), &portid, C.IB_ATTR_SMINFO, 0, 0, srcport)

	C.mad_decode_field(&sminfo[0], C.IB_SMINFO_GUID_F, unsafe.Pointer(&guid))
	C.mad_decode_field(&sminfo[0], C.IB_SMINFO_ACT_F, unsafe.Pointer(&act))
	C.mad_decode_field(&sminfo[0], C.IB_SMINFO_PRIO_F, unsafe.Pointer(&prio))
	C.mad_decode_field(&sminfo[0], C.IB_SMINFO_STATE_F, unsafe.Pointer(&state))

	fmt.Printf("sminfo: sm lid %d sm guid %#16x, activity count %d priority %d state %d %s\n",
		portid.lid, guid, act, prio, state, smStateMap[state])
}
