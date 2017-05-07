/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * InfluxDB client functions
 * TODO: Add support for specifying retention policy
 */
package main

import (
	"log"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

func writeBatch(conf influxdbConf, batch influxdb.BatchPoints) {
	c, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     conf.Url,
		Username: conf.Username,
		Password: conf.Password,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Write(batch); err != nil {
		log.Fatal(err)
	}

	c.Close()
}
