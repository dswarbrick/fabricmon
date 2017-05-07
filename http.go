/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * JSON structs / serialisation for d3.js force graph & HTTP request handling
 */
package main

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type d3Node struct {
	Id       string `json:"id"`
	Desc     string `json:"desc"`
	NodeType int    `json:"nodetype"`
}

type d3Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type d3Topology struct {
	Nodes []d3Node `json:"nodes"`
	Links []d3Link `json:"links"`
}

func marshalTopology(w http.ResponseWriter, req *http.Request, f *Fabric) {
	log.Println(req)

	f.mutex.RLock()
	defer f.mutex.RUnlock()

	jsonBuf, err := json.Marshal(f.topology)
	if err != nil {
		log.Println("JSON error:", err)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")

	gz := gzip.NewWriter(w)
	defer gz.Close()

	n, err := gz.Write(jsonBuf)
	w.Header().Set("Content-Length", strconv.Itoa(n))
}

func serve(listenAddress string, f *Fabric) {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		marshalTopology(w, req, f)
	})

	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %s", err)
	}
}
