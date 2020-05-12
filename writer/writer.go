// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

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
