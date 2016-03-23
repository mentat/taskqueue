package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"
)

type TestHandler struct {
	/*
		An object that implements the HTTP request handler interface
		for testing a web server.
	*/
	Pipeline chan AsyncTask
}

func NewTestHandler() *TestHandler {
	/*
		Create a new Test Handler object.
	*/
	th := &TestHandler{
		Pipeline: make(chan AsyncTask, 10000),
	}
	return th
}

func (t TestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	/*
		"Serve" an HTTP request--and push the task object down a channel.
	*/
	at := AsyncTask{
		URL: req.URL.Path,
	}
	t.Pipeline <- at
}

func setupTestServer(configFile string, port int) (*AMQP, *TestHandler, *net.TCPListener, chan error, error) {
	config, err := ParseConfigFile(configFile)

	if err != nil {
		return nil, nil, nil, nil, err
	}

	server := NewAMQP(config.AmqpServer)
	err = server.Connect()
	if err != nil {
		//t.Fatalf("Could not connect with AMQP server: %s", err)
		return nil, nil, nil, nil, err
	}

	errChan := make(chan error, 1)

	// Spawn goroutines to handle each queue's operation.
	for i := range config.Queues {

		// Go ahead and purge the channel before the test
		server.PurgeQueue(config.Queues[i].Name)

		ch, err := server.GetChannel()

		if err != nil {
			return nil, nil, nil, nil, err
		}

		go readQueue(&config.Queues[i], ch, errChan)
	}

	testHttp := NewTestHandler()
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
	testListener, _ := net.ListenTCP("tcp", tcpAddr)

	// Create a little HTTP server to listen for these callbacks.
	go func() {
		httpServer := http.Server{
			Handler: testHttp,
		}

		httpServer.Serve(testListener)
	}()

	return server, testHttp, testListener, errChan, nil
}

func TestQueueAndWebhook(t *testing.T) {

	port := 9997
	server, handler, listener, errChan, err := setupTestServer("taskqueue.ini", port)
	defer server.Close()
	defer listener.Close()

	if err != nil {
		t.Fatalf(err.Error())
	}

	channel, err := server.GetChannel()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer channel.Close()

	// Now see if we can send some webhooks!
	task1 := AsyncTask{
		URL: fmt.Sprintf("http://localhost:%d/blah", port),
	}

	payLoad, _ := json.Marshal(task1)
	channel.Publish("Queue1", string(payLoad))

	select {
	case err := <-errChan:
		t.Fatalf("Error reading channel: %s", err)
	case msg := <-handler.Pipeline:
		fmt.Printf("Got it: %s", msg.URL)
	}
}

func BenchmarkWebhooksQueue3(b *testing.B) {
	port := 9999 + rand.Intn(100)

	server, handler, listener, errChan, err := setupTestServer("taskqueue.ini", port)

	defer server.Close()
	defer listener.Close()

	if err != nil {
		b.Fatalf(err.Error())
	}

	// Create a channel for publishing messages.
	channel, err := server.GetChannel()
	if err != nil {
		b.Fatalf(err.Error())
	}
	defer channel.Close()

	receiveCount := 0

	fmt.Printf("Bench count is %d\n", b.N)

	for i := 0; i < b.N; i++ {
		// Now see if we can send some webhooks!
		task1 := AsyncTask{
			URL: fmt.Sprintf("http://localhost:%d/blah", port),
		}

		payLoad, _ := json.Marshal(task1)
		err := channel.Publish("Queue3", string(payLoad))
		if err != nil {
			b.Fatalf("Error publishing message: %s", err)
		}

	}

	lastMessageAt := time.Now()

loop:
	for receiveCount != b.N {
		select {
		case err := <-errChan:
			b.Fatalf("Error reading channel: %s", err)
		case <-handler.Pipeline:
			lastMessageAt = time.Now()
			receiveCount++
		default:
			if time.Now().Unix()-lastMessageAt.Unix() > 10 {
				// Check queue
				c, err := channel.CountMessages("Queue3")
				if err != nil {
					fmt.Println("Count fail: ", err)
				} else {
					fmt.Println("Messages in queue is: ", c)
				}
				break loop
			}
		}
	}

	if receiveCount != b.N {
		b.Errorf("Not all of the messages have been received: %d", receiveCount)
	}
}
