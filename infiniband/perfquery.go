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
func getPortCounters(portId *C.ib_portid_t, portNum int, ibmadPort *C.struct_ibmad_port, resetThreshold uint) (map[uint32]interface{}, error) {
	var buf [1024]byte

	counters := make(map[uint32]interface{})

	// PerfMgt ClassPortInfo is a required attribute. See ClassPortInfo, IBTA spec v1.3, table 126.
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

			if float64(counters[field].(uint32)) > (float64(counter.Limit) * float64(resetThreshold) / 100) {
				log.Warnf("Port %d counter %s (%d) exceeds threshold",
					portNum, counter.Name, counters[field])
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
