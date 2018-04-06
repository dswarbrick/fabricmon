// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// InfluxDB client functions.
// TODO: Add support for specifying retention policy.

package influxdb

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/influxdata/influxdb/client/v2"

	"github.com/dswarbrick/fabricmon/config"
	"github.com/dswarbrick/fabricmon/infiniband"
)

type InfluxDBWriter struct {
	Config config.InfluxDBConf
}

func (w *InfluxDBWriter) Receiver(input chan infiniband.Fabric) {
	log.Printf("%#v\n", w.Config)

	for m := range input {
		log.Printf("InfluxDB receiver: %#v\n", m)
	}

	log.Println("InfluxDB receiver input channel closed. Exiting function.")
}

func writeInfluxDB(nodes []infiniband.Node, conf config.InfluxDBConf, caName string, portNum int) {
	// Batch to hold InfluxDB points
	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
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
		tags["guid"] = fmt.Sprintf("%016x", node.GUID)

		for portNum, port := range node.Ports {
			tags["port"] = strconv.Itoa(portNum)

			for counter, value := range port.Counters {
				switch value.(type) {
				case uint32:
					tags["counter"] = infiniband.StdCounterMap[counter]
				case uint64:
					tags["counter"] = infiniband.ExtCounterMap[counter]
				}

				// FIXME: InfluxDB < 1.6 does not support uint64
				// (https://github.com/influxdata/influxdb/pull/8923)
				// Workaround is to either convert to int64 (i.e., truncate to 63 bits).
				if v, ok := value.(uint64); ok {
					fields["value"] = int64(v & 0x7fffffffffffffff)
				}

				if point, err := client.NewPoint("fabricmon_counters", tags, fields, now); err == nil {
					batch.AddPoint(point)
				}
			}
		}
	}

	log.Printf("InfluxDB batch contains %d points\n", len(batch.Points()))
	writeBatch(conf, batch)
}

func writeBatch(conf config.InfluxDBConf, batch client.BatchPoints) {
	client, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     conf.Url,
		Username: conf.Username,
		Password: conf.Password,
	})

	if err != nil {
		log.Print(err)
		return
	}

	if rtt, version, err := client.Ping(0); err == nil {
		log.Printf("InfluxDB (version %s) ping response: %v\n", version, rtt)
	}

	if err := client.Write(batch); err != nil {
		log.Print(err)
	}

	client.Close()
}
