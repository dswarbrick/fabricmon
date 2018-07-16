// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Package forecegraph implements the ForceGraphWriter, which writes the fabric topology to a JSON
// file suitable for use by the d3.js force graph functions.
package forcegraph

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/dswarbrick/fabricmon/infiniband"
)

type d3Node struct {
	ID       string `json:"id"`
	Desc     string `json:"desc"`
	NodeType int    `json:"nodetype"`
	VendorID uint   `json:"vendor_id"`
	DeviceID uint   `json:"device_id"`
}

type d3Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Width  string `json:"link_width"`
	Speed  string `json:"link_speed"`
}

type d3Topology struct {
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
			if err := writeTopology(fg.OutputDir, fabric); err != nil {
				log.WithError(err).Error("cannot marshal fabric to force graph topology")
			}
		}
	}
}

// buildTopology transforms the internal representation of InfiniBand nodes into d3.js nodes and
// links.
func buildTopology(nodes []infiniband.Node) d3Topology {
	topo := d3Topology{}

	for _, node := range nodes {
		d3n := d3Node{
			ID:       fmt.Sprintf("%016x", node.GUID),
			NodeType: node.NodeType,
			Desc:     node.NodeDesc,
			VendorID: node.VendorID,
			DeviceID: node.DeviceID,
		}

		topo.Nodes = append(topo.Nodes, d3n)

		for _, port := range node.Ports {
			if port.RemoteGUID != 0 {
				topo.Links = append(topo.Links, d3Link{
					Source: fmt.Sprintf("%016x", node.GUID),
					Target: fmt.Sprintf("%016x", port.RemoteGUID),
					Width:  port.LinkWidth,
					Speed:  port.LinkSpeed,
				})
			}
		}
	}

	return topo
}

// writeTopology writes a d3.js force graph JSON object file.
func writeTopology(outputDir string, fabric infiniband.Fabric) error {
	// Write d3.js topology to a temporary file, then rename it to target file, to ensure atomic
	// updates and avoid partial reads by clients.
	tempFile, err := ioutil.TempFile(outputDir, ".fabricmon")
	if err != nil {
		return err
	}

	enc := json.NewEncoder(tempFile)
	if err := enc.Encode(buildTopology(fabric.Nodes)); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return err
	}

	tempFile.Close()
	destFile := fmt.Sprintf("%s-%s-p%d.json", fabric.Hostname, fabric.CAName, fabric.SourcePort)

	if err := os.Rename(tempFile.Name(), filepath.Join(outputDir, destFile)); err != nil {
		os.Remove(tempFile.Name())
		return err
	}

	return nil
}
