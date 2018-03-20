// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Config parsing.

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
