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
	Concurrency       int
	Rate              string
	RateDetails       rateSpec
	RetryLimit        int
	MinBackoffSeconds int
	MaxBackoffSeconds int
	MaxDoublings      int
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
	sections := make([]queueConfig, 0, len(sectionNames))

	for i := range sectionNames {
		if sectionNames[i] == "" {
			panic("Nope")
		}

		section := queueConfig{
			Name:              "default",
			Concurrency:       1,
			Rate:              "1/s",
			RetryLimit:        -1,
			MinBackoffSeconds: 0,
			MaxBackoffSeconds: -1,
			MaxDoublings:      -1,
		}

		sections = append(sections, section)
	}

	return config, nil
}
