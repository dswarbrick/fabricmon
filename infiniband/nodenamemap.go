// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Go implementation of InfiniBand node name map lookup functions.

package infiniband

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/fsnotify/fsnotify"
)

const DEFAULT_NODE_NAME_MAP = "/etc/opensm/ib-node-name-map"

var nnMap *NodeNameMap

// The NodeNameMap type holds a mapping of a 64-bit GUID to an InfiniBand node name / description.
type NodeNameMap struct {
	filePath string
	nodes    map[uint64]string
	lock     sync.RWMutex
	watcher  *fsnotify.Watcher
}

// NewNodeNameMap opens and parses the SM node name map, returning a NodeNameMap of GUIDs and their
// node descriptions. The format of the node name map file is described in man page
// ibnetdiscover(8).
func NewNodeNameMap(filePath string) (*NodeNameMap, error) {
	n := &NodeNameMap{filePath: filePath}

	if err := n.reload(); err != nil {
		return n, err
	}

	if watcher, err := fsnotify.NewWatcher(); err == nil {
		n.watcher = watcher
		if err := n.watcher.Add(n.filePath); err != nil {
			slog.Error("cannot add fsnotify watch for node name map", "err", err)
		}
	} else {
		slog.Error("cannot create fsnotify watcher", "err", err)
		return n, err
	}

	go func() {
		for {
			select {
			case event := <-n.watcher.Events:
				// Ignore chmod, everything else requires a reload
				if event.Op^fsnotify.Chmod == 0 {
					break
				}

				slog.Info("node name map watcher event", "event", event.Op)

				if event.Op == fsnotify.Remove {
					if err := n.watcher.Add(n.filePath); err != nil {
						slog.Error("cannot re-add fsnotify watcher for node name map", "err", err)
					}
				} else {
					if err := n.reload(); err != nil {
						slog.Error("failed to reload node name map", "err", err)
					} else {
						slog.Info("node name map reloaded")
					}
				}

			case err := <-n.watcher.Errors:
				if err != nil {
					slog.Error("error watching node name map", "err", err)
				}
			}
		}
	}()

	return n, nil
}

// RemapNodeName attempts to map the specified GUID to a node description from the NodeNameMap. If
// the GUID is not found in the map, the supplied node description is simply returned unmodified.
func (n *NodeNameMap) RemapNodeName(guid uint64, nodeDesc string) string {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if mapDesc, ok := n.nodes[guid]; ok {
		return mapDesc
	}
	return nodeDesc
}

func (n *NodeNameMap) reload() error {
	nodes := make(map[uint64]string)

	file, err := os.Open(n.filePath)
	if err != nil {
		return err
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

	n.lock.Lock()
	n.nodes = nodes
	n.lock.Unlock()

	return nil
}

func init() {
	var err error

	if nnMap, err = NewNodeNameMap(DEFAULT_NODE_NAME_MAP); err != nil {
		// FIXME: since init() functions are called before main(), this can result in
		// inconsistent log formatting, as the default logger is not yet initialised.
		slog.Error("cannot load node name map", "err", err)
	}
}
