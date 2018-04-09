// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package writer

import (
	"github.com/dswarbrick/fabricmon/infiniband"
)

type FMWriter interface {
	Receiver(chan infiniband.Fabric)
}
