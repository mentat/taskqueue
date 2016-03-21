package main

import (
	"fmt"
	"regexp"

	"github.com/go-ini/ini"
)

var amqpServerRe = regexp.MustCompile("amqp://([^:]+):([^@]+)@[^:]+:[0-9]+")

type rateSpec struct {
	Amount            int
	IntervalInSeconds int
	Interval          string
}

type queueConfig struct {
	Name              string
	Concurrency       int    `ini:"concurrency"`
	Rate              string `ini:"rate"`
	RetryLimit        int    `ini:"retry_limit"`
	MinBackoffSeconds int    `ini:"min_backoff_seconds"`
	MaxBackoffSeconds int    `ini:"max_backoff_seconds"`
	MaxDoublings      int    `ini:"max_doublings"`
	RateDetails       rateSpec
}

type ServerConfig struct {
	AmqpServer string
	Queues     []queueConfig
}

func ParseConfigFile(filename string) (*ServerConfig, error) {
	/*
	   Read in the configuration values.
	*/
	cfg, err := ini.Load(filename)

	if err != nil {
		return nil, err
	}

	server := cfg.Section("").Key("server").String()

	if !amqpServerRe.MatchString(server) {
		return nil, fmt.Errorf("AMQP server definition is invalid.")
	}

	config := &ServerConfig{
		AmqpServer: server,
	}

	sectionNames := cfg.SectionStrings()
	sections := make([]queueConfig, 0, len(sectionNames)-1)

	for i := range sectionNames {
		if sectionNames[i] == "DEFAULT" {
			continue
		}

		section := queueConfig{
			Name:              sectionNames[i],
			Concurrency:       1,
			Rate:              "1/s",
			RetryLimit:        -1,
			MinBackoffSeconds: 0,
			MaxBackoffSeconds: -1,
			MaxDoublings:      -1,
		}

		err := cfg.Section(sectionNames[i]).MapTo(&section)
		if err != nil {
			return nil, fmt.Errorf("Configuration format invalid: %s", err)
		}

		sections = append(sections, section)
	}

	config.Queues = sections

	return config, nil
}
