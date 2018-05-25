// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Package config handles the configuration parsing for FabricMon.
package config

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
)

// FabricmonConf is the main configuration struct for FabricMon.
type FabricmonConf struct {
	PollInterval   Duration `toml:"poll_interval"`
	ResetThreshold uint     `toml:"counter_reset_threshold"`
	InfluxDB       []InfluxDBConf
	Logging        LoggingConf
	Topology       TopologyConf
}

// InfluxDBConf holds the configuration values for a single InfluxDB instance.
type InfluxDBConf struct {
	URL      string
	Database string
	Username string
	Password string
}

type LoggingConf struct {
	EnableSyslog bool     `toml:"enable_syslog"`
	LogLevel     LogLevel `toml:"log_level"`
}

type TopologyConf struct {
	Enabled   bool
	OutputDir string `toml:"output_dir"`
}

func (conf *TopologyConf) validate() error {
	if conf.Enabled {
		if err := unix.Access(conf.OutputDir, unix.W_OK); err != nil {
			return fmt.Errorf("Topology output directory: %s", err)
		}
	}

	return nil
}

// Duration is a TOML wrapper type for time.Duration.
// See https://github.com/golang/go/issues/24174.
type Duration time.Duration

// String returns the string representation of the duration.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// UnmarshalText parses a byte slice value into a time.Duration value.
func (d *Duration) UnmarshalText(text []byte) error {
	// Ignore if there is no value set.
	if len(text) == 0 {
		return nil
	}

	// Otherwise parse as a duration formatted string.
	value, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}

	// Set duration and return.
	*d = Duration(value)
	return nil
}

// LogLevel is a TOML wrapper type for logrus.Level.
type LogLevel logrus.Level

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	return logrus.Level(l).String()
}

// UnmarshalText parses a byte slice value into a logrus.Level value.
func (l *LogLevel) UnmarshalText(text []byte) error {
	level, err := logrus.ParseLevel(string(text))

	if err == nil {
		*l = LogLevel(level)
	}

	return err
}

func ReadConfig(configFile string) (FabricmonConf, error) {
	// Defaults
	conf := FabricmonConf{
		PollInterval: Duration(time.Second * 10),
		Logging: LoggingConf {
			LogLevel: LogLevel(logrus.InfoLevel),
		},
		Topology: TopologyConf{},
	}

	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		return conf, fmt.Errorf("Cannot open / parse config file: %s", err)
	}

	if err := conf.Topology.validate(); err != nil {
		return conf, err
	}

	return conf, nil
}
