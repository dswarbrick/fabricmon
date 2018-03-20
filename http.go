// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// JSON structs / serialisation for d3.js force graph & HTTP request handling.

package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// indexHandler renders a simple landing page for browsers that stumble upon the URL
func indexHandler(w http.ResponseWriter, req *http.Request) {
	// The "/" pattern matches everything, so we need to check that we're at the root here.
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

	fm := req.Context().Value("fabricMap").(FabricMap)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<!DOCTYPE html>\n"+
		"<html>\n"+
		"<h1>FabricMon</h1>\n"+
		"<h2>Available Fabrics</h2>\n")

	for caName, _ := range fm {
		for portNum, _ := range fm[caName] {
			fmt.Fprintf(w, "<a href=\"/json/?ca=%s&port=%d\">%s port %d</a>\n",
				caName, portNum, caName, portNum)
		}
	}

	fmt.Fprint(w, "</html>\n")
}

func marshalTopology(w http.ResponseWriter, req *http.Request) {
	var f *Fabric

	query := req.URL.Query()

	if ca, exists := query["ca"]; exists {
		if port, exists := query["port"]; exists {
			portNum, _ := strconv.ParseInt(port[0], 10, 0)
			fm := req.Context().Value("fabricMap").(FabricMap)
			f = fm[ca[0]][int(portNum)]
		}
	}

	if f == nil {
		http.NotFound(w, req)
		return
	}

	f.mutex.RLock()
	defer f.mutex.RUnlock()

	jsonBuf, err := json.Marshal(f.topology)
	if err != nil {
		log.Println("JSON error:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBuf)
}

// middleware adds the fabric pointer to the request context and wraps the ResponseWriter in a gzip
// handler.
func middleware(next http.Handler, fm FabricMap) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r)
		ctx := context.WithValue(r.Context(), "fabricMap", fm)

		// Keep XHR requests happy
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gz := gzip.NewWriter(w)
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			next.ServeHTTP(gzipResponseWriter{gz, w}, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func serve(listenAddr string, fm FabricMap) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/json/", marshalTopology)

	log.Fatal(http.ListenAndServe(listenAddr, middleware(mux, fm)))
}
