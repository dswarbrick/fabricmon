// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Low-level bit operations.

package infiniband

import (
	"testing"
)

func TestMaxPow2Divisor(t *testing.T) {
	if maxPow2Divisor(4+2+1, 2+1) != 2 {
		t.Fail()
	}

	if maxPow2Divisor(8, 4+2+1) != 0 {
		t.Fail()
	}

	if maxPow2Divisor(8+4+1, 2+1) != 1 {
		t.Fail()
	}
}
