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
		Pipeline: make(chan AsyncTask, 100),
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

func setupTestServer(configFile string, port int) (*AMQP, *TestHandler, *net.TCPListener, error) {
	config, err := ParseConfigFile(configFile)

	if err != nil {
		return nil, nil, nil, err
	}

	server := NewAMQP(config.AmqpServer)
	err = server.Connect()
	if err != nil {
		//t.Fatalf("Could not connect with AMQP server: %s", err)
		return nil, nil, nil, err
	}

	errChan := make(chan error)

	// Spawn goroutines to handle each queue's operation.
	for i := range config.Queues {

		// Go ahead and purge all configured queues to ensure we are getting
		// fresh messages.  Only do this if queue exists.
		if _, err := server.CountMessages(config.Queues[i].Name); err == nil {
			server.PurgeQueue(config.Queues[i].Name)
		}

		delivery, err := server.ConsumeQueue(config.Queues[i].Name)
		if err != nil {
			//t.Fatalf("Could not read queue: %s %s", config.Queues[i].Name, err)
			return nil, nil, nil, fmt.Errorf("Cannot consume queue: %s", err)
		}

		go readQueue(&config.Queues[i], delivery, errChan)
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

	return server, testHttp, testListener, nil
}

func TestQueueAndWebhook(t *testing.T) {

	port := 9997
	server, handler, listener, err := setupTestServer("taskqueue.ini", port)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Now see if we can send some webhooks!
	task1 := AsyncTask{
		URL: fmt.Sprintf("http://localhost:%d/blah", port),
	}

	payLoad, _ := json.Marshal(task1)
	server.Publish("Queue1", string(payLoad))

	select {
	case msg := <-handler.Pipeline:
		fmt.Printf("Got it: %s", msg.URL)
	}

	listener.Close()
	server.Close()
}

func BenchmarkWebhooks(b *testing.B) {
	port := 9999 + rand.Intn(100)
	server, handler, listener, err := setupTestServer("taskqueue.ini", port)
	if err != nil {
		b.Fatalf(err.Error())
	}

	receiveCount := 0

	fmt.Printf("Bench count is %d\n", b.N)

	for i := 0; i < b.N; i++ {
		// Now see if we can send some webhooks!
		task1 := AsyncTask{
			URL: fmt.Sprintf("http://localhost:%d/blah", port),
		}

		payLoad, _ := json.Marshal(task1)
		server.Publish("Queue1", string(payLoad))

	}

	lastMessageAt := time.Now()

loop:
	for receiveCount != b.N {
		select {
		case <-handler.Pipeline:
			lastMessageAt = time.Now()
			receiveCount++
			//fmt.Println("REQ")
		default:
			if time.Now().Unix()-lastMessageAt.Unix() > 3 {
				break loop
			}

		}
	}

	if receiveCount != b.N {
		b.Errorf("Not all of the messages have been received: %d", receiveCount)
	}

	listener.Close()
	server.Close()
}
