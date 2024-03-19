// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package infiniband

// #cgo CFLAGS: -I/usr/include/infiniband
// #cgo LDFLAGS: -libnetdisc
// #include <umad.h>
// #include <ibnetdisc.h>
// #include <iba/ib_types.h>
import "C"

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	log "github.com/sirupsen/logrus"
)

type HCA struct {
	Name string

	// umad_ca_t contains an array of pointers - associated memory must be freed with
	// umad_release_ca(umad_ca_t *ca)
	umad_ca *C.umad_ca_t
}

func (h *HCA) NetDiscover(output chan Fabric, mkey uint64, resetThreshold uint) {
	var (
		totalNodes, totalPorts int
	)

	mgmt_classes := [...]C.int{C.IB_SMI_CLASS, C.IB_SA_CLASS, C.IB_PERFORMANCE_CLASS}

	hostname, _ := os.Hostname()
	start := time.Now()

	// Iterate over CA's umad_port array
	for _, umad_port := range h.umad_ca.ports {
		// umad_ca.ports may contain noncontiguous umad_port pointers
		if umad_port == nil {
			continue
		}

		portNum := int(umad_port.portnum)
		linkLayer := C.GoString(&umad_port.link_layer[0])
		portLog := log.WithFields(log.Fields{"ca": h.Name, "port": portNum})

		if linkLayer != "InfiniBand" && linkLayer != "IB" {
			portLog.Debugf("Skipping port with unsupported link layer %q", linkLayer)
			continue
		}

		portLog.Debug("Polling port")

		// ibnd_config_t specifies max hops, timeout, max SMPs etc
		config := C.ibnd_config_t{flags: C.IBND_CONFIG_MLX_EPI, mkey: C.uint64_t(mkey)}

		// NOTE: Under ibsim, this will fail after a certain number of iterations with a
		// mad_rpc_open_port() error (presumably due to a resource leak in ibsim).
		// ibnd_fabric_t *ibnd_discover_fabric(char *ca_name, int ca_port, ib_portid_t *from, ibnd_config_t *config)
		fabric, err := C.ibnd_discover_fabric(&h.umad_ca.ca_name[0], umad_port.portnum, nil, &config)

		if err != nil {
			portLog.WithError(err).Error("Unable to discover fabric")
			continue
		}

		// Open MAD port, which is needed for getting port counters.
		// struct ibmad_port *mad_rpc_open_port(char *dev_name, int dev_port, int *mgmt_classes, int num_classes)
		mad_port := C.mad_rpc_open_port(&h.umad_ca.ca_name[0], umad_port.portnum, &mgmt_classes[0], C.int(len(mgmt_classes)))

		if mad_port != nil {
			nodes := walkFabric(fabric, mad_port, resetThreshold)
			C.mad_rpc_close_port(mad_port)

			totalNodes += len(nodes)

			for _, n := range nodes {
				totalPorts += len(n.Ports)
			}

			if output != nil {
				output <- Fabric{
					Hostname:   hostname,
					CAName:     h.Name,
					SourcePort: portNum,
					Nodes:      nodes,
				}
			}
		} else {
			portLog.Error("Unable to open MAD port")
		}

		C.ibnd_destroy_fabric(fabric)
	}

	log.WithFields(log.Fields{
		"time":  time.Since(start),
		"nodes": totalNodes,
		"ports": totalPorts},
	).Info("NetDiscover completed")
}

func (h *HCA) Release() {
	// Free associated memory from pointers in umad_ca_t.ports
	if C.umad_release_ca(h.umad_ca) < 0 {
		log.Error("umad_release_ca: ", h.umad_ca)
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

		log.WithFields(log.Fields{
			"ca":          C.GoString(&ca.ca_name[0]),
			"type":        C.GoString(&ca.ca_type[0]),
			"ports":       ca.numports,
			"firmware":    C.GoString(&ca.fw_ver[0]),
			"hardware":    C.GoString(&ca.hw_ver[0]),
			"node_guid":   fmt.Sprintf("%#016x", ntohll(uint64(ca.node_guid))),
			"system_guid": fmt.Sprintf("%#016x", ntohll(uint64(ca.system_guid))),
		}).Info("Found HCA")

		hcas[i] = HCA{
			Name:    caName,
			umad_ca: &ca,
		}
	}

	return hcas
}

type ibndNode struct {
	ibnd_node *C.struct_ibnd_node
	log       *log.Entry
}

// getPortCounters retrieves all counters for a specific port.
// Note: In PortCounters, PortCountersExtended, PortXmitDataSL, and PortRcvDataSL, components that
// represent Data (e.g. PortXmitData and PortRcvData) indicate octets divided by 4 rather than just
// octets.
func (n *ibndNode) getPortCounters(portId *C.ib_portid_t, portNum int, ibmadPort *C.struct_ibmad_port, resetThreshold uint) (map[uint32]interface{}, error) {
	var buf [1024]byte

	counters := make(map[uint32]interface{})
	portLog := n.log.WithFields(log.Fields{"port": portNum})

	// PerfMgt ClassPortInfo is a required attribute. See ClassPortInfo, IBTA spec v1.3, table 126.
	pmaBuf := C.pma_query_via(unsafe.Pointer(&buf), portId, C.int(portNum), PMA_TIMEOUT, C.CLASS_PORT_INFO, ibmadPort)

	if pmaBuf == nil {
		return counters, fmt.Errorf("CLASS_PORT_INFO query failed!")
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

			if float64(counters[field].(uint32)) > (float64(counter.Limit) * float64(resetThreshold) / 100) {
				portLog.WithFields(log.Fields{
					"counter": counter.Name,
					"value":   counters[field],
				}).Warn("Counter exceeds threshold")

				selMask |= counter.Select
			}
		}

		if selMask > 0 {
			var pc [1024]byte

			resetLog := portLog.WithFields(log.Fields{"select_mask": fmt.Sprintf("%#x", selMask)})
			resetLog.Warn("Resetting counters")

			if C.performance_reset_via(unsafe.Pointer(&pc), portId, C.int(portNum), C.uint(selMask), PMA_TIMEOUT, C.IB_GSI_PORT_COUNTERS, ibmadPort) == nil {
				resetLog.Error("performance_reset_via failed")
			}
		}
	}

	if (capMask&C.IB_PM_EXT_WIDTH_SUPPORTED == 0) && (capMask&C.IB_PM_EXT_WIDTH_NOIETF_SUP == 0) {
		// TODO: Fetch standard data / packet counters if extended counters are not supported
		// (pre-QDR hardware).
		portLog.Warn("Port does not support extended counters")
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

func (n *ibndNode) guid() uint64 {
	return uint64(n.ibnd_node.guid)
}

func (n *ibndNode) guidString() string {
	return fmt.Sprintf("%#016x", n.ibnd_node.guid)
}

func (n *ibndNode) nodeDesc() string {
	return C.GoString(&n.ibnd_node.nodedesc[0])
}

// simpleNode returns a Node structure, containing only safe Go types, suitable for asynchronous
// access, even if the original fabric pointers have been freed.
func (n *ibndNode) simpleNode() Node {
	if n.ibnd_node == nil {
		return Node{}
	}

	node := Node{
		GUID:     n.guid(),
		NodeType: int(n.ibnd_node._type),
		NodeDesc: nnMap.RemapNodeName(n.guid(), n.nodeDesc()),
		VendorID: uint(C.mad_get_field(unsafe.Pointer(&n.ibnd_node.info), 0, C.IB_NODE_VENDORID_F)),
		DeviceID: uint(C.mad_get_field(unsafe.Pointer(&n.ibnd_node.info), 0, C.IB_NODE_DEVID_F)),
	}

	return node
}

func (n *ibndNode) walkPorts(mad_port *C.struct_ibmad_port, resetThreshold uint) []Port {
	var portid C.ib_portid_t

	n.log.WithFields(log.Fields{
		"node_type": n.ibnd_node._type,
		"num_ports": n.ibnd_node.numports,
	}).Debug("Walking ports for node")

	ports := make([]Port, n.ibnd_node.numports+1)

	C.ib_portid_set(&portid, C.int(n.ibnd_node.smalid), 0, 0)

	// node.ports is an array of ports, indexed by port number:
	//   ports[1] == port 1,
	//   ports[2] == port 2,
	//   etc...
	// Any port in the array MAY BE NIL! Most notably, non-switches have no port zero, therefore
	// ports[0] == nil for those nodes!
	arrayPtr := uintptr(unsafe.Pointer(n.ibnd_node.ports))

	for portNum := 0; portNum <= int(n.ibnd_node.numports); portNum++ {
		var (
			info         *[C.IB_SMP_DATA_SIZE]C.uchar
			linkSpeedExt uint
		)

		portLog := n.log.WithFields(log.Fields{"port": portNum})

		// Get pointer to port struct at portNum array offset
		pp := *(**C.ibnd_port_t)(unsafe.Pointer(arrayPtr + unsafe.Sizeof(arrayPtr)*uintptr(portNum)))
		if pp == nil {
			continue
		}

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
		if n.ibnd_node._type == C.IB_NODE_SWITCH {
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

		portLog.WithFields(log.Fields{
			"port_state": PortStateToStr(uint(portState)),
			"phys_state": PortPhysStateToStr(uint(physState)),
			"link_width": myPort.LinkWidth,
			"link_speed": myPort.LinkSpeed,
		}).Debugf("Port info")

		// Remote port may be nil if port state is polling / armed.
		rp := pp.remoteport

		if rp != nil {
			myPort.RemoteGUID = uint64(rp.node.guid)
			myPort.RemoteNodeDesc = C.GoString(&rp.node.nodedesc[0])

			// Port counters will only be fetched if port is ACTIVE + LINKUP
			if (portState == C.IB_LINK_ACTIVE) && (physState == C.IB_PORT_PHYS_STATE_LINKUP) {
				// Determine max width supported by both ends
				maxWidth := maxPow2Divisor(
					uint(C.mad_get_field(unsafe.Pointer(&pp.info), 0, C.IB_PORT_LINK_WIDTH_SUPPORTED_F)),
					uint(C.mad_get_field(unsafe.Pointer(&rp.info), 0, C.IB_PORT_LINK_WIDTH_SUPPORTED_F)))

				if uint(linkWidth) != maxWidth {
					portLog.Warn("Link width is not the max width supported by both ports")
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

				if counters, err := n.getPortCounters(&portid, portNum, mad_port, resetThreshold); err == nil {
					myPort.Counters = counters
				} else {
					portLog.WithError(err).Error("Cannot get counters for port")
				}
			}
		}

		ports[portNum] = myPort
	}

	return ports
}

func walkFabric(fabric *C.struct_ibnd_fabric, mad_port *C.struct_ibmad_port, resetThreshold uint) []Node {
	nodes := make([]Node, 0)

	for node := fabric.nodes; node != nil; node = node.next {
		n := ibndNode{ibnd_node: node}
		n.log = log.WithFields(log.Fields{
			"node_desc": nnMap.RemapNodeName(n.guid(), n.nodeDesc()),
			"node_guid": n.guidString(),
		})

		myNode := n.simpleNode()

		if n.ibnd_node._type == C.IB_NODE_SWITCH {
			myNode.Ports = n.walkPorts(mad_port, resetThreshold)
		}

		nodes = append(nodes, myNode)
	}

	return nodes
}
