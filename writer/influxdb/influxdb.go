// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// InfluxDB client functions.
// TODO: Add support for specifying retention policy.

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
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     w.Config.Url,
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

	// TODO: Break this out to a separate, unexported method.
	for fabric := range input {
		batch, err := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  w.Config.Database,
			Precision: "s",
		})
		if err != nil {
			log.Error(err)
			return
		}

		tags := map[string]string{
			"host":     fabric.Hostname,
			"hca":      fabric.CAName,
			"src_port": strconv.Itoa(fabric.SourcePort),
		}

		fields := map[string]interface{}{}
		now := time.Now()

		for _, node := range fabric.Nodes {
			if node.NodeType != 2 { // Switch
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
						// FIXME: InfluxDB < 1.6 does not support uint64
						// (https://github.com/influxdata/influxdb/pull/8923)
						// Workaround is to convert to int64 (i.e., truncate to 63 bits).
						fields["value"] = int64(v & 0x7fffffffffffffff)
					}

					if point, err := client.NewPoint("fabricmon_counters", tags, fields, now); err == nil {
						batch.AddPoint(point)
					}
				}
			}
		}

		log.Infof("InfluxDB batch contains %d points\n", len(batch.Points()))

		if err := c.Write(batch); err != nil {
			log.Error(err)
		}
	}

	log.Debug("InfluxDBWriter input channel closed.")
	c.Close()
	log.Debug("InfluxDBWriter out.")
}
