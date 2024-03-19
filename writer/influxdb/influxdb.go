// Copyright 2017-20 Daniel Swarbrick. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package influxdb implements the InfluxDBWriter, which writes InfiniBand performance counters to
// one or more configured InfluxDB backends.
package influxdb

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/influxdata/influxdb/client/v2"

	"github.com/dswarbrick/fabricmon/config"
	"github.com/dswarbrick/fabricmon/infiniband"
)

const (
	// TODO: Consider making this configurable
	measurementName = "fabricmon_counters"
)

type InfluxDBWriter struct {
	config config.InfluxDBConf
}

func NewInfluxDBWriter(config config.InfluxDBConf) *InfluxDBWriter {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &InfluxDBWriter{config: config}
}

// TODO: Rename this to something more descriptive (and which is not so easily confused with method
// receivers).
func (w *InfluxDBWriter) Receiver(input chan infiniband.Fabric) {
	// InfluxDB client opens connections on demand, so we can preemptively create it here.
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:      w.config.URL,
		Username:  w.config.Username,
		Password:  w.config.Password,
		Timeout:   w.config.Timeout,
		UserAgent: "FabricMon",
	})

	if err != nil {
		slog.Error("cannot init InfluxDB client", "err", err)
		return
	}

	if rtt, version, err := c.Ping(0); err == nil {
		slog.Info("InfluxDB ping reply", "version", version, "rtt", rtt)
	}

	// Loop indefinitely until input chan closed.
	for fabric := range input {
		if batch, err := w.makeBatch(fabric); err == nil {
			slog.Debug("InfluxDB batch created",
				"hca", fabric.CAName,
				"port", fabric.SourcePort,
				"points", len(batch.Points()))

			if err := c.Write(batch); err != nil {
				slog.Error("InfluxDB batch write error", "err", err)
			}
		} else {
			slog.Error("InfluxDB batch creation error", "err", err)
		}
	}

	slog.Debug("InfluxDBWriter input channel closed. Closing InfluxDB client connections.")
	c.Close()
}

func (w *InfluxDBWriter) makeBatch(fabric infiniband.Fabric) (client.BatchPoints, error) {
	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        w.config.Database,
		RetentionPolicy: w.config.RetentionPolicy,
		Precision:       "s",
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
		tags["node_desc"] = node.NodeDesc

		for portNum, port := range node.Ports {
			tags["port"] = strconv.Itoa(portNum)

			if port.RemoteGUID != 0 {
				tags["remote_guid"] = fmt.Sprintf("%016x", port.RemoteGUID)
				tags["remote_node_desc"] = port.RemoteNodeDesc
			} else {
				delete(tags, "remote_guid")
				delete(tags, "remote_node_desc")
			}

			for counter, value := range port.Counters {
				switch v := value.(type) {
				case uint32:
					tags["counter"] = infiniband.StdCounterMap[counter].Name
					fields["value"] = int64(v)
				case uint64:
					tags["counter"] = infiniband.ExtCounterMap[counter].Name
					// InfluxDB Client docs erroneously claim that "uint64 data type is supported
					// if your server is version 1.4.0 or greater."
					// In fact, it has been decided that InfluxDB 1.x will never support uint64:
					// https://github.com/influxdata/influxdb/pull/8923
					// Workaround is to convert to int64 (i.e., truncate to 63 bits).
					fields["value"] = int64(v & 0x7fffffffffffffff)
				default:
					continue
				}

				if point, err := client.NewPoint(measurementName, tags, fields, now); err == nil {
					batch.AddPoint(point)
				}
			}
		}
	}

	return batch, nil
}
