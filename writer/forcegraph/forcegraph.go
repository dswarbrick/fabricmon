// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// JSON structs / serialisation for d3.js force graph

package forcegraph

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	log "github.com/sirupsen/logrus"

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

type D3Topology struct {
	Nodes []d3Node `json:"nodes"`
	Links []d3Link `json:"links"`
}

type ForceGraphWriter struct {
	OutputDir string
}

// TODO: Rename this to something more descriptive (and which is not so easily confused with method
// receivers).
func (fg *ForceGraphWriter) Receiver(input chan infiniband.Fabric) {
	for fabric := range input {

		if fg.OutputDir != "" {
			filename := fmt.Sprintf("%s-%s-p%d.json",
				fabric.Hostname, fabric.CAName, fabric.SourcePort)

			buf := makeD3(fabric.Nodes)

			// FIXME: This should write to a temporary file and perform an atomic move / rename,
			// in case something is reading the .json file at the same time as we write to it.
			if err := ioutil.WriteFile(path.Join(fg.OutputDir, filename), buf, 0644); err != nil {
				log.WithError(err).Error("Cannot write d3.js JSON topology")
			}
		}
	}
}

// makeD3 transforms the internal representation of InfiniBand nodes into d3.js nodes and links,
// and returns marshalled JSON.
func makeD3(nodes []infiniband.Node) []byte {
	nnMap, _ := infiniband.NewNodeNameMap()

	topo := D3Topology{}

	for _, node := range nodes {
		d3n := d3Node{
			ID:       fmt.Sprintf("%016x", node.GUID),
			NodeType: node.NodeType,
			Desc:     nnMap.RemapNodeName(node.GUID, node.NodeDesc),
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
		log.WithError(err).Error("JSON error")
		return nil
	}

	return jsonBuf
}
