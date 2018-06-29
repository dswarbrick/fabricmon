// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// cgo wrapper around libibumad / libibnetdiscover.
// Note: Due to the usual permissions on /dev/infiniband/umad*, this will probably need to be
// executed as root.

// Package fabricmon is an InfiniBand fabric monitor daemon.
//
package main

import (
	"fmt"
	"log/syslog"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"

	log "github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/dswarbrick/fabricmon/config"
	"github.com/dswarbrick/fabricmon/infiniband"
	"github.com/dswarbrick/fabricmon/version"
	"github.com/dswarbrick/fabricmon/writer"
	"github.com/dswarbrick/fabricmon/writer/forcegraph"
	"github.com/dswarbrick/fabricmon/writer/influxdb"
)

// router duplicates a Fabric struct received via channel and outputs it to multiple receiver
// channels.
func router(input chan infiniband.Fabric, writers []writer.FabricWriter) {
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

	log.Debug("Router input channel closed. Exiting function.")
}

func main() {
	var (
		configFile = kingpin.Flag("config", "Path to config file.").Default("fabricmon.conf").String()
		daemonize  = kingpin.Flag("daemonize", "Run forever, fetching counters periodically.").Default("true").Bool()
	)

	kingpin.Parse()

	conf, err := config.ReadConfig(*configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialise umad library (also required in order to run under ibsim).
	if infiniband.UmadInit() < 0 {
		fmt.Println("Error initialising umad library. Exiting.")
		os.Exit(1)
	}

	hcas := infiniband.GetCAs()

	if len(hcas) == 0 {
		fmt.Println("No HCAs found in system. Exiting.")
		os.Exit(1)
	}

	if conf.Logging.EnableSyslog {
		log.SetFormatter(&log.TextFormatter{DisableColors: true, DisableTimestamp: true})

		if hook, err := lSyslog.NewSyslogHook("", "", syslog.LOG_INFO, ""); err == nil {
			log.AddHook(hook)
		}
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	}

	log.SetLevel(log.Level(conf.Logging.LogLevel))
	log.Info("FabricMon ", version.Info())

	// Channel to signal goroutines that we are shutting down.
	shutdownChan := make(chan bool)

	// Setup signal handler to catch SIGINT, SIGTERM.
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)
	go func() {
		s := <-sigChan
		log.WithFields(log.Fields{"signal": s}).Debug("Shutting down due to signal.")
		close(shutdownChan)
	}()

	// Initialize empty slice to hold writers
	writers := make([]writer.FabricWriter, 0)

	if conf.Topology.Enabled {
		writers = append(writers, &forcegraph.ForceGraphWriter{OutputDir: conf.Topology.OutputDir})
	}

	// First sweep.
	for _, hca := range hcas {
		hca.NetDiscover(nil)
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
				ticker.Stop()
				for _, hca := range hcas {
					hca.NetDiscover(splitter)
				}
				intvl := int64(time.Duration(conf.PollInterval))
				r := rand.Int63n(intvl/5) - intvl/10
				ticker = time.NewTicker(time.Duration(conf.PollInterval) + time.Duration(r))
			case <-shutdownChan:
				log.Debug("Shutdown received in polling loop.")
				break Loop
			}
		}

		close(splitter)
	}

	log.Debug("Cleaning up")

	// Free associated memory from pointers in umad_ca_t.ports
	for _, hca := range hcas {
		hca.Release()
	}

	infiniband.UmadDone()
}
