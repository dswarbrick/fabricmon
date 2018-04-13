// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #include <mad.h>
// #include <ibnetdisc.h>
import "C"

import (
	"fmt"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

// getPortCounters retrieves all counters for a specific port.
// Note: In PortCounters, PortCountersExtended, PortXmitDataSL, and PortRcvDataSL, components that
// represent Data (e.g. PortXmitData and PortRcvData) indicate octets divided by 4 rather than just
// octets.
func getPortCounters(portId *C.ib_portid_t, portNum int, ibmadPort *C.struct_ibmad_port) (map[uint32]interface{}, error) {
	var buf [1024]byte

	counters := make(map[uint32]interface{})

	// PerfMgt ClassPortInfo is a required attribute
	pmaBuf := C.pma_query_via(unsafe.Pointer(&buf), portId, C.int(portNum), PMA_TIMEOUT, C.CLASS_PORT_INFO, ibmadPort)

	if pmaBuf == nil {
		return counters, fmt.Errorf("ERROR: Port %d CLASS_PORT_INFO query failed!", portNum)
	}

	// Keep capMask in network byte order for easier bitwise operations with capabilities contants.
	capMask := htons(uint16(C.mad_get_field(unsafe.Pointer(&buf), 0, C.IB_CPI_CAPMASK_F)))

	// Fetch standard (32 bit (or less)) counters
	pmaBuf = C.pma_query_via(unsafe.Pointer(&buf), portId, C.int(portNum), PMA_TIMEOUT, C.IB_GSI_PORT_COUNTERS, ibmadPort)

	if pmaBuf != nil {
		var selMask uint32

		// Iterate over standard counters
		for field, counter := range StdCounterMap {
			if (field == C.IB_PC_XMT_WAIT_F) && (capMask&C.IB_PM_PC_XMIT_WAIT_SUP == 0) {
				continue // Counter not supported
			}

			counters[field] = uint32(C.mad_get_field(unsafe.Pointer(&buf), 0, field))

			// FIXME: Honour the counter_reset_threshold value in config
			if float64(counters[field].(uint32)) > (float64(counter.Limit) * 0.01) {
				selMask |= counter.Select
			}
		}

		if selMask > 0 {
			var pc [1024]byte

			log.WithFields(log.Fields{"select_mask": fmt.Sprintf("%#x", selMask)}).
				Warn("Resetting counters")

			if C.performance_reset_via(unsafe.Pointer(&pc), portId, C.int(portNum), C.uint(selMask), PMA_TIMEOUT, C.IB_GSI_PORT_COUNTERS, ibmadPort) == nil {
				log.Error("performance_reset_via failed")
			}
		}
	}

	if (capMask&C.IB_PM_EXT_WIDTH_SUPPORTED == 0) && (capMask&C.IB_PM_EXT_WIDTH_NOIETF_SUP == 0) {
		// TODO: Fetch standard data / packet counters if extended counters are not supported
		// (pre-QDR hardware).
		log.WithFields(log.Fields{"port": portNum}).Warn("Port does not support extended counters")
		return counters, nil
	}

	// Fetch extended (64 bit) counters
	pmaBuf = C.pma_query_via(unsafe.Pointer(&buf), portId, C.int(portNum), PMA_TIMEOUT, C.IB_GSI_PORT_COUNTERS_EXT, ibmadPort)

	if pmaBuf != nil {
		for field, _ := range ExtCounterMap {
			counters[field] = uint64(C.mad_get_field64(unsafe.Pointer(&buf), 0, field))
		}
	}

	return counters, nil
}

func walkPorts(node *C.struct_ibnd_node, mad_port *C.struct_ibmad_port) []Port {
	var portid C.ib_portid_t

	log.Debugf("Node type: %d, node descr: %s, num. ports: %d, node GUID: %#016x",
		node._type, nnMap.RemapNodeName(uint64(node.guid), C.GoString(&node.nodedesc[0])),
		node.numports, node.guid)

	ports := make([]Port, node.numports+1)

	C.ib_portid_set(&portid, C.int(node.smalid), 0, 0)

	// node.ports is an array of ports, indexed by port number:
	//   ports[1] == port 1,
	//   ports[2] == port 2,
	//   etc...
	// Any port in the array MAY BE NIL! Most notably, non-switches have no port zero, therefore
	// ports[0] == nil for those nodes!
	arrayPtr := uintptr(unsafe.Pointer(node.ports))

	for portNum := 0; portNum <= int(node.numports); portNum++ {
		var (
			info         *[C.IB_SMP_DATA_SIZE]C.uchar
			linkSpeedExt uint
		)

		// Get pointer to port struct at portNum array offset
		pp := *(**C.ibnd_port_t)(unsafe.Pointer(arrayPtr + unsafe.Sizeof(arrayPtr)*uintptr(portNum)))

		myPort := Port{GUID: uint64(pp.guid)}

		portState := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_STATE_F)
		physState := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_PHYS_STATE_F)

		// C14-24.2.1 states that a down port allows for invalid data to be returned for all
		// PortInfo components except PortState and PortPhysicalState.
		if portState == C.IB_LINK_DOWN {
			ports[portNum] = myPort
			continue
		}

		linkWidth := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_WIDTH_ACTIVE_F)
		myPort.LinkWidth = LinkWidthToStr(uint(linkWidth))

		// Check for extended speed support
		if node._type == C.IB_NODE_SWITCH {
			info = &(*(**C.ibnd_port_t)(unsafe.Pointer(arrayPtr))).info
		} else {
			info = &pp.info
		}

		capMask := htonl(uint32(C.mad_get_field(unsafe.Pointer(info), 0, C.IB_PORT_CAPMASK_F)))
		if capMask&C.IB_PORT_CAP_HAS_EXT_SPEEDS != 0 {
			linkSpeedExt = uint(C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_SPEED_EXT_ACTIVE_F))
		}

		if linkSpeedExt > 0 {
			myPort.LinkSpeed = LinkSpeedExtToStr(linkSpeedExt)
		} else {
			fdr10 := C.mad_get_field(unsafe.Pointer(&pp.ext_info), 0, C.IB_MLNX_EXT_PORT_LINK_SPEED_ACTIVE_F) & C.FDR10

			if fdr10 != 0 {
				myPort.LinkSpeed = "FDR10"
			} else {
				linkSpeed := C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_SPEED_ACTIVE_F)
				myPort.LinkSpeed = LinkSpeedToStr(uint(linkSpeed))
			}
		}

		log.Debugf("Port %d, port state: %s, phys state: %s, link width: %s, link speed: %s",
			portNum,
			PortStateToStr(uint(portState)), PortPhysStateToStr(uint(physState)),
			myPort.LinkWidth, myPort.LinkSpeed)

		// Remote port may be nil if port state is polling / armed.
		rp := pp.remoteport

		if rp != nil {
			myPort.RemoteGUID = uint64(rp.node.guid)

			// Port counters will only be fetched if port is ACTIVE + LINKUP
			if (portState == C.IB_LINK_ACTIVE) && (physState == C.IB_PORT_PHYS_STATE_LINKUP) {
				// Determine max width supported by both ends
				maxWidth := maxPow2Divisor(
					uint(C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_WIDTH_SUPPORTED_F)),
					uint(C.mad_get_field(unsafe.Pointer(&rp.info), 0, C.IB_PORT_LINK_WIDTH_SUPPORTED_F)))

				if uint(linkWidth) != maxWidth {
					log.Warnf("Port %d link width is not the max width supported by both ports",
						portNum)
				}

				// Determine max speed supported by both ends
				// TODO: Check for possible FDR10 support (ext_info IB_MLNX_EXT_PORT_LINK_SPEED_SUPPORTED_F)
				// TODO: Check for possible extended speed (info IB_PORT_LINK_SPEED_EXT_SUPPORTED_F)
				/*
					maxSpeed := maxPow2Divisor(
						uint(C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_SPEED_SUPPORTED_F)),
						uint(C.mad_get_field(unsafe.Pointer(&rp.info), 0, C.IB_PORT_LINK_SPEED_SUPPORTED_F)))

					if uint(linkSpeed) != maxSpeed {
						log.Warnf("Port %d link speed is not the max speed supported by both ports",
							portNum)
					}
				*/

				if counters, err := getPortCounters(&portid, portNum, mad_port); err == nil {
					myPort.Counters = counters
				} else {
					log.WithError(err).WithFields(log.Fields{"port": portNum}).
						Error("Cannot get counters for port")
				}
			}
		}

		ports[portNum] = myPort
	}

	return ports
}
