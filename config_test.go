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

	if len(config.Queues) != 2 {
		t.Fatalf("Invalid parse of config queues.")
	}

	if config.Queues[1].Name != "Queue2" {
		t.Fatalf("Invalid parse of config queues name: %v", config.Queues[1])
	}

	if config.Queues[1].Concurrency != 3 {
		t.Fatalf("Invalid parse of config queues detail: %v", config.Queues[1])
	}
}
