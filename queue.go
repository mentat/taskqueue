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

func readQueue(config *queueConfig, messages <-chan amqp.Delivery, err chan error) {

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

			// Check if we've retried this before:
			rt, ok := retryData.Get(d.MessageId)
			if ok {
				// Check for ETA, don't execute early.
				if rt.ETA > time.Now().Unix() {
					// Sleep a little bit to prevent a busy loop.
					time.Sleep(100 * time.Millisecond)
					d.Nack(false, true)
					return
				}
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
				if rt.MaxRetries != -1 && rt.CurrentRetries > rt.MaxRetries {
					Error.Printf("Task %s exceeded maximum retries and was cancelled.", d.MessageId)
					d.Nack(false, false)
				} else {
					d.Nack(false, true)
				}

			} else {
				// Task processed OK.
				d.Ack(false)
			}

		}(&msg, errorChan)
	}

}
