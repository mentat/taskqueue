package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

func readQueue(config *queueConfig, messages <-chan amqp.Delivery, err chan error) {

	// Create semaphore for concurrency
	sem := make(chan bool, config.Concurrency)
	errorChan := make(chan error, config.Concurrency)

	for msg := range messages {

		// We've received a message, lock a space in semaphore
		sem <- true

		go func(d *amqp.Delivery, ec chan error) {
			defer func() { <-sem }()

			// Task data is JSON, decode it to AsyncTask struct.
			var task AsyncTask
			err := json.Unmarshal(d.Body, task)
			if err != nil {
				ec <- err
				fmt.Println("Could not unmarshal data.", err.Error())
				return
			}

			// Give the webhook a long timeout...
			timeout := time.Duration(60 * 60 * 6 * time.Second)
			client := http.Client{
				Timeout: timeout,
			}

			resp, err := client.Post(task.URL, "application/json",
				strings.NewReader(task.Payload))

			if err != nil {
				// Some kind of error, retry
			} else if resp.StatusCode != 200 {
				// Some kind of error, retry
			} else {
				// Task processed OK.
				d.Ack(false)
			}

		}(&msg, errorChan)
	}

}
