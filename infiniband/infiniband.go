// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #include <ibnetdisc.h>
import "C"

type Fabric struct {
	Hostname   string
	CAName     string
	SourcePort int
	Nodes      []Node
}

type Node struct {
	GUID     uint64
	NodeType int
	NodeDesc string
	VendorID uint16
	DeviceID uint16
	Ports    []Port
}

type Port struct {
	GUID       uint64
	RemoteGUID uint64
	Counters   map[uint32]interface{}
}

// Standard (32-bit) counters and their display names
// TODO: Implement warnings and / or automatically reset counters when they are close to reaching
// 	     their maximum permissible value (according to IBTA spec).
var StdCounterMap = map[uint32]string{
	C.IB_PC_ERR_SYM_F:        "SymbolErrorCounter",
	C.IB_PC_LINK_RECOVERS_F:  "LinkErrorRecoveryCounter",
	C.IB_PC_LINK_DOWNED_F:    "LinkDownedCounter",
	C.IB_PC_ERR_RCV_F:        "PortRcvErrors",
	C.IB_PC_ERR_PHYSRCV_F:    "PortRcvRemotePhysicalErrors",
	C.IB_PC_ERR_SWITCH_REL_F: "PortRcvSwitchRelayErrors",
	C.IB_PC_XMT_DISCARDS_F:   "PortXmitDiscards",
	C.IB_PC_ERR_XMTCONSTR_F:  "PortXmitConstraintErrors",
	C.IB_PC_ERR_RCVCONSTR_F:  "PortRcvConstraintErrors",
	C.IB_PC_ERR_LOCALINTEG_F: "LocalLinkIntegrityErrors",
	C.IB_PC_ERR_EXCESS_OVR_F: "ExcessiveBufferOverrunErrors",
	C.IB_PC_VL15_DROPPED_F:   "VL15Dropped",
	C.IB_PC_XMT_WAIT_F:       "PortXmitWait", // Requires cap mask IB_PM_PC_XMIT_WAIT_SUP
}

// Extended (64-bit) counters and their display names.
var ExtCounterMap = map[uint32]string{
	C.IB_PC_EXT_XMT_BYTES_F: "PortXmitData",
	C.IB_PC_EXT_RCV_BYTES_F: "PortRcvData",
	C.IB_PC_EXT_XMT_PKTS_F:  "PortXmitPkts",
	C.IB_PC_EXT_RCV_PKTS_F:  "PortRcvPkts",
	C.IB_PC_EXT_XMT_UPKTS_F: "PortUnicastXmitPkts",
	C.IB_PC_EXT_RCV_UPKTS_F: "PortUnicastRcvPkts",
	C.IB_PC_EXT_XMT_MPKTS_F: "PortMulticastXmitPkts",
	C.IB_PC_EXT_RCV_MPKTS_F: "PortMulticastRcvPkts",
}

var portStates = [...]string{
	"No state change", // Valid only on Set() port state
	"Down",            // Includes failed links
	"Initialize",
	"Armed",
	"Active",
}

var portPhysStates = [...]string{
	"No state change", // Valid only on Set() port state
	"Sleep",
	"Polling",
	"Disabled",
	"PortConfigurationTraining",
	"LinkUp",
	"LinkErrorRecovery",
	"Phy Test",
}
