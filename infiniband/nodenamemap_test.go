// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

package infiniband

import (
	"reflect"
	"testing"
)

var (
	// Expected nodes after parsing node name map
	nodes = map[uint64]string {
		0xb7c31c3b29d0c791: "ibsw1(root-sw)",
		0xa31de6b2f83b0a91: "ibsw2",
		0x4878ef07ca6bf2a0: "\"sw1 - root\"",
		0x9cf5e55c63d7a4a3: "\"sw1 #root#\"",
	}
)

func TestRemapNodeName(t *testing.T) {
	nnMap, err := NewNodeNameMap("testdata/ib-node-name-map")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(nnMap.nodes, nodes) {
		t.Fatal("Parsed map does not match expected")
	}

	if nnMap.RemapNodeName(0xb7c31c3b29d0c791, "") != "ibsw1(root-sw)" {
		t.Fail()
	}

	if nnMap.RemapNodeName(0xb7c31c3b29d0c791, "foo") != "ibsw1(root-sw)" {
		t.Fail()
	}

	if nnMap.RemapNodeName(0x123, "non-existent") != "non-existent" {
		t.Fail()
	}
}
