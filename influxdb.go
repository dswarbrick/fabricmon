// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// InfluxDB client functions.
// TODO: Add support for specifying retention policy.

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
		log.Print(err)
	}

	if err := c.Write(batch); err != nil {
		log.Print(err)
	}

	c.Close()
}
