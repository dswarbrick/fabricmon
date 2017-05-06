/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * Functions analogous to libosmcomp
 */
package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const DEFAULT_NODE_NAME_MAP = "/etc/opensm/ib-node-name-map"

type NodeNameMap map[uint64]string

// NewNodeNameMap opens and parses the SM node name map, returning a NodeNameMap of GUIDs and their
// node descriptions. The format of the node name map file is described in man page
// ibnetdiscover(8).
func NewNodeNameMap() (NodeNameMap, error) {
	nodes := make(map[uint64]string)

	file, err := os.Open(DEFAULT_NODE_NAME_MAP)
	if err != nil {
		return nodes, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Tokenize line, honouring quoted strings
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.FieldsFunc(line, f)
		if len(fields) < 2 || strings.HasPrefix(fields[1], "#") {
			continue
		}

		guid, err := strconv.ParseUint(fields[0], 0, 64)
		if err != nil {
			continue
		}

		nodes[guid] = fields[1]
	}

	return nodes, nil
}

// remapNodeName attempts to map the specified GUID to a node description from the NodeNameMap. If
// the GUID is not found in the map, the supplied node description is simply returned unmodified.
func (n NodeNameMap) remapNodeName(guid uint64, nodeDesc string) string {
	if mapDesc, ok := n[guid]; ok {
		return mapDesc
	}
	return nodeDesc
}
