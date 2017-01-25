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

	ch, err := server.GetChannel()
	if err != nil {
		t.Fatalf("Cannot get channel: %s", err)
	}

	c, err := ch.ConsumeQueue("MyStuff")
	if err != nil {
		t.Fatalf("Cannot consume Rabbit queue: %s", err)
	}

	ch.Publish("MyStuff", "message1")
	ch.Publish("MyStuff", "message2")
	ch.Publish("MyStuff", "message3")

	count := 0
	for msg := range c {
		fmt.Println(string(msg.Body))
		count++
		if count == 3 {
			break
		}
	}
	ch.Close()
	server.Close()
}
