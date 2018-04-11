// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// cgo wrapper around libibumad / libibnetdiscover.
// Note: Due to the usual permissions on /dev/infiniband/umad*, this will probably need to be
// executed as root.

// Package FabricMon is an InfiniBand fabric monitor daemon.
//
package main

// #cgo CFLAGS: -I/usr/include/infiniband
// #include <umad.h>
// #include <ibnetdisc.h>
import "C"

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/dswarbrick/fabricmon/config"
	"github.com/dswarbrick/fabricmon/infiniband"
	"github.com/dswarbrick/fabricmon/version"
	"github.com/dswarbrick/fabricmon/writer"
	"github.com/dswarbrick/fabricmon/writer/forcegraph"
	"github.com/dswarbrick/fabricmon/writer/influxdb"
)

type Fabric struct {
	mutex      sync.RWMutex
	ibndFabric *C.struct_ibnd_fabric
	ibmadPort  *C.struct_ibmad_port
	topology   forcegraph.D3Topology
}

// FabricMap is a two-dimensional map holding the Fabric struct for each HCA / port pair.
type FabricMap map[string]map[int]*Fabric

// router duplicates a Fabric struct received via channel and outputs it to multiple receiver
// channels.
func router(input chan infiniband.Fabric, writers []writer.FMWriter) {
	outputs := make([]chan infiniband.Fabric, len(writers))

	// Create output channels for workers, and start worker goroutine
	for i, w := range writers {
		outputs[i] = make(chan infiniband.Fabric)
		go w.Receiver(outputs[i])
	}

	for fabric := range input {
		for _, c := range outputs {
			c <- fabric
		}
	}

	// Close output channels
	for _, c := range outputs {
		close(c)
	}

	log.Println("Router input channel closed. Exiting function.")
}

func main() {
	var (
		configFile = kingpin.Flag("config", "Path to config file.").Default("fabricmon.conf").String()
		jsonDir    = kingpin.Flag("json-dir", "Output directory for JSON topologies.").Default("./").String()
		daemonize  = kingpin.Flag("daemonize", "Run forever, fetching counters periodically.").Default("true").Bool()
	)

	kingpin.Parse()

	conf, err := config.ReadConfig(*configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialise umad library (also required in order to run under ibsim)
	// NOTE: ibsim indicates that FabricMon is not "disconnecting" when it exits - resource leak?
	if C.umad_init() < 0 {
		fmt.Println("Error initialising umad library. Exiting.")
		os.Exit(1)
	}

	caNames := infiniband.UmadGetCANames()

	if len(caNames) == 0 {
		fmt.Println("No HCAs found in system. Exiting.")
		os.Exit(1)
	}

	log.Println("FabricMon", version.Info())

	// umad_ca_t contains an array of pointers - associated memory must be freed with
	// umad_release_ca(umad_ca_t *ca)
	umad_ca_list := make([]C.umad_ca_t, len(caNames))

	for i, caName := range caNames {
		var ca C.umad_ca_t

		ca_name := C.CString(caName)
		C.umad_get_ca(ca_name, &ca)
		C.free(unsafe.Pointer(ca_name))

		log.Printf("Found CA %s (%s) with %d ports, firmware version: %s, hardware version: %s, "+
			"node GUID: %#016x, system GUID: %#016x\n",
			C.GoString(&ca.ca_name[0]), C.GoString(&ca.ca_type[0]), ca.numports,
			C.GoString(&ca.fw_ver[0]), C.GoString(&ca.hw_ver[0]),
			infiniband.Ntohll(uint64(ca.node_guid)), infiniband.Ntohll(uint64(ca.system_guid)))

		umad_ca_list[i] = ca
	}

	// Channel to signal goroutines that we are shutting down.
	shutdownChan := make(chan bool)

	// Setup signal handler to catch SIGINT, SIGTERM.
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)
	go func() {
		s := <-sigChan
		log.Printf("Caught signal: %s. Shutting down.", s)
		close(shutdownChan)
	}()

	// Initialize writers slice with just the d3.js ForceGraphWriter
	writers := []writer.FMWriter{&forcegraph.ForceGraphWriter{OutputDir: *jsonDir}}

	// First sweep.
	for _, ca := range umad_ca_list {
		infiniband.CADiscoverFabric(ca, nil)
	}

	if *daemonize {
		for _, c := range conf.InfluxDB {
			w := &influxdb.InfluxDBWriter{Config: c}
			writers = append(writers, w)
		}

		// FIXME: Move this outside of daemonize if-block
		splitter := make(chan infiniband.Fabric)
		go router(splitter, writers)

		ticker := time.NewTicker(time.Duration(conf.PollInterval))
		defer ticker.Stop()

	Loop:
		// Loop indefinitely, scanning fabrics every tick.
		for {
			select {
			case <-ticker.C:
				for _, ca := range umad_ca_list {
					infiniband.CADiscoverFabric(ca, splitter)
				}
			case <-shutdownChan:
				log.Println("Shutdown received in polling loop.")
				break Loop
			}
		}

		close(splitter)
	}

	log.Println("Cleaning up")

	// Free associated memory from pointers in umad_ca_t.ports
	for _, ca := range umad_ca_list {
		C.umad_release_ca(&ca)
	}

	infiniband.UmadDone()

	// TODO: Re-enable these
	// Start HTTP server to serve JSON for d3.js (WIP)
	// serve(conf.BindAddress, fabrics)
}
