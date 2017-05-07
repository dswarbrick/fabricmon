/*
 * FabricMon - an InfiniBand fabric monitor daemon.
 * Copyright 2017 Daniel Swarbrick
 *
 * Config parsing
 */
package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type influxdbConf struct {
	Url      string
	Database string
	Username string
	Password string
}

type fabricmonConf struct {
	BindAddress string
	InfluxDB    influxdbConf
}

func readConfig(configFile string) (fabricmonConf, error) {
	var conf fabricmonConf

	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		return conf, fmt.Errorf("Cannot open / parse config file: %s", err)
	}

	return conf, nil
}
