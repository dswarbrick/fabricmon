// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// InfluxDB client functions.
// TODO: Add support for specifying retention policy.

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

func writeInfluxDB(nodes []Node, conf influxdbConf, caName string, portNum int) {
	// Batch to hold InfluxDB points
	batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  conf.Database,
		Precision: "s",
	})
	if err != nil {
		return
	}

	hostname, _ := os.Hostname()
	tags := map[string]string{"host": hostname, "hca": caName, "src_port": strconv.Itoa(portNum)}
	fields := map[string]interface{}{}
	now := time.Now()

	for _, node := range nodes {
		tags["guid"] = fmt.Sprintf("%016x", node.guid)

		for portNum, port := range node.ports {
			tags["port"] = strconv.Itoa(portNum)

			for counter, value := range port.counters {
				switch value.(type) {
				case uint32:
					tags["counter"] = stdCounterMap[counter]
					fields["value"] = value
				case uint64:
					tags["counter"] = extCounterMap[counter]

					// FIXME: InfluxDB < 1.6 does not support uint64
					// (https://github.com/influxdata/influxdb/pull/8923)
					// Workaround is to either convert to int64 (i.e., truncate to 63 bits),
					fields["value"] = int64(value.(uint64))
				}

				if point, err := influxdb.NewPoint("fabricmon_counters", tags, fields, now); err == nil {
					batch.AddPoint(point)
				}
			}
		}
	}

	fmt.Printf("InfluxDB batch contains %d points\n", len(batch.Points()))
	writeBatch(conf, batch)
}

func writeBatch(conf influxdbConf, batch influxdb.BatchPoints) {
	c, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     conf.Url,
		Username: conf.Username,
		Password: conf.Password,
	})
	if err != nil {
		log.Print(err)
	}

	if err := c.Write(batch); err != nil {
		log.Print(err)
	}

	c.Close()
}
