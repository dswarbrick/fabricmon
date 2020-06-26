// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package writer defines the FabricWriter interface type, which all FabricMon writers must
// implement in order to receive fabric topologies and counters. What each writer does with that
// information is dependent on the individual writer.
package writer

import (
	"github.com/dswarbrick/fabricmon/infiniband"
)

// FabricWriter defines the interface type that all FabricMon writers must implement.
type FabricWriter interface {
	Receiver(chan infiniband.Fabric)
}
