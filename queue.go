package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type ackMessage struct {
	Msg *amqp.Delivery
	Ack bool
	Requeue bool
}

type RetryData struct {
	MaxRetries     int64
	CurrentRetries int64
	LastBackoff    int
	ETA            int64 // Time as posixtime
}

type ThreadSafeMap struct {
	data  map[string]RetryData
	mutex sync.RWMutex
}

func NewThreadSafeMap() *ThreadSafeMap {
	tsm := &ThreadSafeMap{
		data: make(map[string]RetryData),
	}
	return tsm
}

func (m *ThreadSafeMap) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.data, key)
}

func (m *ThreadSafeMap) Set(key string, data RetryData) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = data
}

func (m *ThreadSafeMap) Get(key string) (data RetryData, ok bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	data, ok = m.data[key]
	return
}

func calculateEta(now *time.Time, rt *RetryData, task *AsyncTask, config *queueConfig) {
	/*
	   Calculate the new ETA for the task. Update the Posix time on the RT data.
	*/
	if rt.LastBackoff == 0 {
		rt.LastBackoff = config.MinBackoffSeconds
	} else if rt.CurrentRetries < int64(config.MaxDoublings) {
		if (rt.LastBackoff * 2) < config.MaxBackoffSeconds {
			rt.LastBackoff = rt.LastBackoff * 2
		}
		// Otherwise leave the backoff as is
	}
	rt.ETA = now.Unix() + int64(rt.LastBackoff)
}

func readTask(d *amqp.Delivery, retryData *ThreadSafeMap, config *queueConfig,
	ec chan error, sem chan bool, ack chan ackMessage) {
	/*
		Read a task and dispatch it's webhook.
	*/
	defer func() { <-sem }()

	// Task data is JSON, decode it to AsyncTask struct.
	var task AsyncTask
	err := json.Unmarshal(d.Body, &task)
	if err != nil {
		ec <- err
		fmt.Println("Could not unmarshal data.", err.Error())
		return
	}

	// Current time of system
	now := time.Now()

	// Check if we've retried this before:
	rt, retryDataOk := retryData.Get(d.MessageId)
	if !retryDataOk {
		rt.CurrentRetries = 0
		rt.MaxRetries = task.MaxRetries
		rt.ETA = task.ETA
	}

	// Check for ETA, don't execute early.
	if rt.ETA > now.Unix() {
		// Sleep a little bit to prevent a busy loop.
		time.Sleep(100 * time.Millisecond)
		ack <- ackMessage{
		   Ack: false,
		   Msg: d,
	   	}
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
		fmt.Printf("ERR1: %s\n", err)
		// Some kind of error, retry, don't count this
		// as a retry since it is probably a service issue
		d.Nack(false, true)
		ack <- ackMessage{
		   Ack: false,
		   Requeue: true,
		   Msg: d,
	   	}
	} else if resp.StatusCode != 200 {
		fmt.Printf("ERR2\n")
		// Some kind of error, retry
		rt.CurrentRetries++
		// Check to see if we have exceeded the max retries.
		if rt.MaxRetries != -1 && rt.CurrentRetries > rt.MaxRetries {
			retryData.Delete(d.MessageId)
			Error.Printf("Task %s exceeded maximum retries and was cancelled.", d.MessageId)
			ack <- ackMessage{
			   Ack: false,
			   Requeue: false,
			   Msg: d,
		   	}
		} else {
			calculateEta(&now, &rt, &task, config)
			retryData.Set(d.MessageId, rt)
			ack <- ackMessage{
			   Ack: false,
			   Requeue: true,
			   Msg: d,
		   	}
		}

	} else {
		// Task processed OK.

		ack <- ackMessage{
			Ack: true,
			Msg: d,
		}
	}

}

func readQueue(config *queueConfig, channel *AMQPChannel, ec chan error) {
	/*
	   Read from a RabbitMQ queue and spawn a goroutine to handle the
	   dispatch of the HTTP request.
	*/

	// Close the AMQP channel after the goroutine is done.
	defer channel.Close()

	messages, err := channel.ConsumeQueue(config.Name)
	if err != nil {
		ec <- err
		return
	}

	// Create semaphore for concurrency
	sem := make(chan bool, config.Concurrency)

	errorChan := make(chan error, config.Concurrency)
	defer close(errorChan)

	ackChan := make(chan ackMessage, config.Concurrency)
	defer close(ackChan)

	// Initialize the last fill time to zero-value
	lastFillAt := time.Time{}

	// Create a thread-safe map to store task retry data.
	retryData := NewThreadSafeMap()

	// Constantly listen for messages on RabbitMQ queue.
	for msg := range messages {

		// We've received a message, lock a space in semaphore
		sem <- true

		// Enforce queue fill rate...
		if lastFillAt.IsZero() {
			lastFillAt = time.Now()
		} else {
			now := time.Now()
			offsetInMillis := (now.UnixNano() - lastFillAt.UnixNano()) / 100000

			fillRate := config.RateDetails.FillRateInMillis

			// The sleeper must awaken...
			if offsetInMillis < fillRate {
				time.Sleep(time.Duration(fillRate-offsetInMillis) * time.Millisecond)
			}
			// Set the new last at.
			lastFillAt = time.Now()
		}

		// Read the task and run the webhook
		go readTask(&msg, retryData, config, errorChan, sem, ackChan)

		select {
		case ack := <- ackChan:
			if ack.Ack {
				ack.Msg.Ack(false)
			} else {
				ack.Msg.Nack(false, ack.Requeue)
			}
		case err := <- errorChan:
			ec <- err
			return
		default:
			// Allow queue to continue...
		}
	}
}
