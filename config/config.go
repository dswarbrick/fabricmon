// Copyright 2017-18 Daniel Swarbrick. All rights reserved.
// Use of this source code is governed by a GPL license that can be found in the LICENSE file.

// Package config handles the configuration parsing for FabricMon.
package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// FabricmonConf is the main configuration struct for FabricMon.
type FabricmonConf struct {
	PollInterval   time.Duration `yaml:"poll_interval"`
	ResetThreshold uint          `yaml:"counter_reset_threshold"`
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
	EnableSyslog bool     `yaml:"enable_syslog"`
	LogLevel     LogLevel `yaml:"log_level"`
}

type TopologyConf struct {
	Enabled   bool
	OutputDir string `yaml:"output_dir"`
}

func (conf *TopologyConf) validate() error {
	if conf.Enabled {
		if err := unix.Access(conf.OutputDir, unix.W_OK); err != nil {
			return fmt.Errorf("Topology output directory: %s", err)
		}
	}

	return nil
}

// LogLevel is a wrapper type for logrus.Level.
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

func ReadConfig(configFile string) (*FabricmonConf, error) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}

	// Defaults
	conf := &FabricmonConf{
		PollInterval: time.Second * 10,
		Logging: LoggingConf{
			LogLevel: LogLevel(logrus.InfoLevel),
		},
	}

	if err := yaml.UnmarshalStrict(content, conf); err != nil {
		return nil, err
	}

	if err := conf.Topology.validate(); err != nil {
		return nil, err
	}

	return conf, nil
}
