// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Low-level bit operations.

package infiniband

import (
	"testing"
)

var (
	flsTest   = [...]uint{0, 1, 0x80000000}
	flsResult = [...]uint{0, 1, 32}
)

func TestFls(t *testing.T) {
	for i, x := range flsTest {
		if Fls(x) != flsResult[i] {
			t.Fatalf("Fls(%d) != %d", Fls(x), flsResult[i])
		}
	}
}
