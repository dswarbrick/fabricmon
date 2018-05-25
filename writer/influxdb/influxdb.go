// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.
//
// TODO: Add support for specifying retention policy.

// Package influxdb implements the InfluxDBWriter, which writes InfiniBand performance counters to
// one or more configured InfluxDB backends.
package influxdb

import (
	"fmt"
	"strconv"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	log "github.com/sirupsen/logrus"

	"github.com/dswarbrick/fabricmon/config"
	"github.com/dswarbrick/fabricmon/infiniband"
)

type InfluxDBWriter struct {
	Config config.InfluxDBConf
}

// TODO: Rename this to something more descriptive (and which is not so easily confused with method
// receivers).
func (w *InfluxDBWriter) Receiver(input chan infiniband.Fabric) {
	// InfluxDB client opens connections on demand, so we can preemptively create it here.
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     w.Config.URL,
		Username: w.Config.Username,
		Password: w.Config.Password,
	})

	if err != nil {
		log.Error(err)
		return
	}

	if rtt, version, err := c.Ping(0); err == nil {
		log.WithFields(log.Fields{"version": version, "rtt": rtt}).Infof("InfluxDB ping reply")
	}

	for fabric := range input {
		if batch, err := w.makeBatch(fabric); err == nil {
			log.Infof("InfluxDB batch contains %d points", len(batch.Points()))

			if err := c.Write(batch); err != nil {
				log.Error(err)
			}
		} else {
			log.Error(err)
		}
	}

	log.Debug("InfluxDBWriter input channel closed. Closing InfluxDB client connections.")
	c.Close()
}

func (w *InfluxDBWriter) makeBatch(fabric infiniband.Fabric) (client.BatchPoints, error) {
	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  w.Config.Database,
		Precision: "s",
	})

	if err != nil {
		return batch, err
	}

	tags := map[string]string{
		"host":     fabric.Hostname,
		"hca":      fabric.CAName,
		"src_port": strconv.Itoa(fabric.SourcePort),
	}

	fields := map[string]interface{}{}
	now := time.Now()

	for _, node := range fabric.Nodes {
		if node.NodeType != infiniband.IB_NODE_SWITCH {
			continue
		}

		tags["guid"] = fmt.Sprintf("%016x", node.GUID)

		for portNum, port := range node.Ports {
			tags["port"] = strconv.Itoa(portNum)

			for counter, value := range port.Counters {
				switch v := value.(type) {
				case uint32:
					tags["counter"] = infiniband.StdCounterMap[counter].Name
					fields["value"] = int64(v)
				case uint64:
					tags["counter"] = infiniband.ExtCounterMap[counter].Name
					// InfluxDB Client docs erroneously claim that "uint64 data type is
					// supported if your server is version 1.4.0 or greater."
					// In fact, uint64 support will not land until InfluxDB 1.6.
					// (https://github.com/influxdata/influxdb/pull/8923)
					// Workaround is to convert to int64 (i.e., truncate to 63 bits).
					fields["value"] = int64(v & 0x7fffffffffffffff)
				default:
					continue
				}

				if point, err := client.NewPoint("fabricmon_counters", tags, fields, now); err == nil {
					batch.AddPoint(point)
				}
			}
		}
	}

	return batch, nil
}
