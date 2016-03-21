package main

import (
	"fmt"
	"testing"
)

func TestAMQPConnect(t *testing.T) {

	server := NewAMQP("amqp://guest:guest@localhost:5672/")
	err := server.Connect()
	if err != nil {
		t.Fatalf("Cannot connect to Rabbit: %s", err)
	}

	c, err := server.ConsumeQueue("MyStuff")
	if err != nil {
		t.Fatalf("Cannot consume Rabbit queue: %s", err)
	}

	server.Publish("MyStuff", "message1")
	server.Publish("MyStuff", "message2")
	server.Publish("MyStuff", "message3")

	count := 0
	for msg := range c {
		fmt.Println(string(msg.Body))
		count++
		if count == 3 {
			break
		}
	}
	server.Close()
}
