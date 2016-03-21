package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type TestHandler struct {
	Pipeline chan AsyncTask
}

func NewTestHandler() *TestHandler {
	th := &TestHandler{
		Pipeline: make(chan AsyncTask),
	}
	return th
}

func (t TestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Printf(req.URL.Path)
	at := AsyncTask{
		URL: req.URL.Path,
	}
	t.Pipeline <- at
}

func TestQueueAndWebhook(t *testing.T) {
	config, err := ParseConfigFile("taskqueue.ini")

	if err != nil {
		t.Fatalf("Cannot parse config: %s", err)
	}

	RabbitServer := NewAMQP(config.AmqpServer)
	err = RabbitServer.Connect()
	if err != nil {
		t.Fatalf("Could not connect with AMQP server: %s", err)
	}

	errChan := make(chan error)

	// Spawn goroutines to handle each queue's operation.
	for i := range config.Queues {

		delivery, err := RabbitServer.ConsumeQueue(config.Queues[i].Name)
		if err != nil {
			t.Fatalf("Could not read queue: %s %s", config.Queues[i].Name, err)
		}

		go readQueue(&config.Queues[i], delivery, errChan)
	}

	testHttp := NewTestHandler()

	// Create a little HTTP server to listen for these callbacks.
	go func() {
		httpServer := http.Server{
			Addr:    ":9999",
			Handler: testHttp,
		}
		httpServer.ListenAndServe()
	}()

	// Now see if we can send some webhooks!
	task1 := AsyncTask{
		URL: "http://localhost:9999/blah",
	}

	payLoad, _ := json.Marshal(task1)
	RabbitServer.Publish("Queue1", string(payLoad))

	select {
	case msg := <-testHttp.Pipeline:
		fmt.Printf("Got it: %s", msg.URL)
	}

}
