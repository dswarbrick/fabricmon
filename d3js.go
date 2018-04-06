// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// JSON structs / serialisation for d3.js force graph

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/dswarbrick/fabricmon/infiniband"
)

type d3Node struct {
	ID       string `json:"id"`
	Desc     string `json:"desc"`
	NodeType int    `json:"nodetype"`
	VendorID uint16 `json:"vendor_id"`
	DeviceID uint16 `json:"device_id"`
}

type d3Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type d3Topology struct {
	Nodes []d3Node `json:"nodes"`
	Links []d3Link `json:"links"`
}

// makeD3 transforms the internal representation of InfiniBand nodes into d3.js nodes and links,
// and returns marshalled JSON.
func makeD3(nodes []infiniband.Node) []byte {
	nnMap, _ := NewNodeNameMap()

	topo := d3Topology{}

	for _, node := range nodes {
		d3n := d3Node{
			ID:       fmt.Sprintf("%016x", node.GUID),
			NodeType: node.NodeType,
			Desc:     nnMap.remapNodeName(node.GUID, node.NodeDesc),
			VendorID: node.VendorID,
			DeviceID: node.DeviceID,
		}

		topo.Nodes = append(topo.Nodes, d3n)

		for _, port := range node.Ports {
			if port.RemoteGUID != 0 {
				topo.Links = append(topo.Links, d3Link{
					fmt.Sprintf("%016x", node.GUID),
					fmt.Sprintf("%016x", port.RemoteGUID),
				})
			}
		}
	}

	jsonBuf, err := json.Marshal(topo)
	if err != nil {
		log.Println("JSON error:", err)
		return nil
	}

	return jsonBuf
}

func writeD3JSON(filename string, nodes []infiniband.Node) {
	buf := makeD3(nodes)

	if err := ioutil.WriteFile(filename, buf, 0644); err != nil {
		log.Println("ERROR: Cannot write d3.js JSON topology:", err)
	}
}
