package main

import "testing"

func TestConfig(t *testing.T) {
	config, err := ParseConfigFile("taskqueue.ini")

	if err != nil {
		t.Fatalf("Cannot parse config: %s", err)
	}

	if config.AmqpServer != "amqp://guest:guest@localhost:5672/" {
		t.Fatalf("Invalid parse of config.")
	}

	if len(config.Queues) != 3 {
		t.Fatalf("Invalid parse of config queues.")
	}

	if config.Queues[0].Rate != "1/s" {
		t.Fatalf("Invalid rate parse: %v", config.Queues[0])
	}

	if config.Queues[0].RateDetails.Amount != 1.0 {
		t.Fatalf("Invalid rate parse: %v", config.Queues[0])
	}

	if config.Queues[0].RateDetails.Interval != "S" {
		t.Fatalf("Invalid rate parse: %v", config.Queues[0])
	}

	if config.Queues[0].RateDetails.FillRateInMillis != 1000 {
		t.Fatalf("Invalid rate parse: %v", config.Queues[0])
	}

	if config.Queues[1].Name != "Queue2" {
		t.Fatalf("Invalid parse of config queues name: %v", config.Queues[1])
	}

	if config.Queues[1].Concurrency != 3 {
		t.Fatalf("Invalid parse of config queues detail: %v", config.Queues[1])
	}

	if config.Queues[1].RateDetails.Amount != 21.5 {
		t.Fatalf("Invalid rate parse A: %v", config.Queues[1])
	}

	if config.Queues[1].RateDetails.Interval != "M" {
		t.Fatalf("Invalid rate parse I: %v", config.Queues[1])
	}

	if config.Queues[1].RateDetails.FillRateInMillis != 2790 {
		t.Fatalf("Invalid rate parse IIS: %v", config.Queues[1])
	}
}
