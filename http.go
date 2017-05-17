/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * JSON structs / serialisation for d3.js force graph & HTTP request handling
 */
package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
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

// indexHandler renders a simple landing page for browsers that stumble upon the URL
func indexHandler(w http.ResponseWriter, req *http.Request) {
	// The "/" pattern matches everything, so we need to check that we're at the root here.
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

	fmt.Fprint(w, "<!DOCTYPE html>\n"+
		"<html>\n"+
		"<h1>FabricMon</h1>\n"+
		"<h2>Available Fabrics</h2>\n"+
		"<a href=\"/json/\">default</a>\n"+
		"</html>\n")
}

func marshalTopology(w http.ResponseWriter, req *http.Request) {
	f := req.Context().Value("fabric").(*Fabric)

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

// addContext adds the fabric pointer to the request context
func addContext(next http.Handler, fabric *Fabric) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r)
		ctx := context.WithValue(r.Context(), "fabric", fabric)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func serve(listenAddress string, f *Fabric) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/json/", marshalTopology)

	log.Fatal(http.ListenAndServe(listenAddress, addContext(mux, f)))
}
