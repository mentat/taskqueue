package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-ini/ini"
)

var (
	amqpServerRe = regexp.MustCompile("amqp://([^:]+):([^@]+)@[^:]+:[0-9]+")
	rateSpecRe   = regexp.MustCompile("([.0-9]+)/([msh])")
)

type rateSpec struct {
	Amount           float64
	FillRateInMillis int64
	Interval         string
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

// ServerConfig -
type ServerConfig struct {
	ServerConnectString string
	Backend             string
	Tombstone           rateSpec
	Queues              []queueConfig
}

func parseRateSpec(spec string) (*rateSpec, error) {
	results := rateSpecRe.FindStringSubmatch(spec)
	if len(results) != 3 {
		return nil, fmt.Errorf("Invalid rate specification.")
	}

	// Err is ignored here due to regex constraint
	rateMultiplier, _ := strconv.ParseFloat(results[1], 64)

	rs := &rateSpec{
		Amount:   rateMultiplier,
		Interval: strings.ToUpper(results[2]),
	}

	// Convert to millisecond per task fill rate
	switch rs.Interval {
	case "S":
		rs.FillRateInMillis = int64(1000 / rs.Amount)
	case "M":
		rs.FillRateInMillis = int64(1000 / (rs.Amount / 60))
	case "H":
		rs.FillRateInMillis = int64(1000 / (rs.Amount / 60 / 60))
	}

	return rs, nil

}

// ParseConfigFile -
func ParseConfigFile(filename string) (*ServerConfig, error) {
	/*
	   Read in the configuration values.
	*/
	cfg, err := ini.Load(filename)

	if err != nil {
		return nil, err
	}

	server := cfg.Section("").Key("server").String()
	backend := cfg.Section("").Key("backend").String()
	tombstoneString := cfg.Section("").Key("tombstone_delay").String()
	tombstone, err := parseRateSpec(tombstoneString)

	if err != nil {
		return nil, fmt.Errorf("Tombstone spec is invalid")
	}

	if backend == "amqp" && !amqpServerRe.MatchString(server) {
		return nil, fmt.Errorf("AMQP server definition is invalid")
	}

	config := &ServerConfig{
		ServerConnectString: server,
		Backend:             backend,
		Tombstone:           *tombstone,
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

		if !rateSpecRe.MatchString(section.Rate) {
			return nil, fmt.Errorf("Queue rate specification is invalid.")
		}

		rs, err := parseRateSpec(section.Rate)
		if err != nil {
			return nil, err
		}

		section.RateDetails = *rs

		sections = append(sections, section)
	}

	config.Queues = sections

	return config, nil
}
