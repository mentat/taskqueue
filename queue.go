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

func readQueue(config *queueConfig, messages <-chan amqp.Delivery, err chan error) {
	/*
	   Read from a RabbitMQ queue and spawn a goroutine to handle the
	   dispatch of the HTTP request.
	*/
	// Create semaphore for concurrency
	sem := make(chan bool, config.Concurrency)
	errorChan := make(chan error, config.Concurrency)

	lastFillAt := time.Time{}

	retryData := NewThreadSafeMap()

	for msg := range messages {

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
		}

		// We've received a message, lock a space in semaphore
		sem <- true

		go func(d *amqp.Delivery, ec chan error) {
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
				d.Nack(false, true)
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
				// Some kind of error, retry, don't count this
				// as a retry since it is probably a service issue
				d.Nack(false, true)
			} else if resp.StatusCode != 200 {
				// Some kind of error, retry
				rt.CurrentRetries++
				// Check to see if we have exceeded the max retries.
				if rt.MaxRetries != -1 && rt.CurrentRetries > rt.MaxRetries {
					retryData.Delete(d.MessageId)
					Error.Printf("Task %s exceeded maximum retries and was cancelled.", d.MessageId)
					d.Nack(false, false)
				} else {
					calculateEta(&now, &rt, &task, config)
					retryData.Set(d.MessageId, rt)
					d.Nack(false, true)
				}

			} else {
				// Task processed OK.
				d.Ack(false)
			}

		}(&msg, errorChan)
	}

}
