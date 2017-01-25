package main

import "fmt"

// Delivery -
type Delivery struct {
	Body      []byte
	MessageID string
}

// Channel --
type Channel interface {
	Publish(queueName, body string) error
	CountMessages(queueName string) (int64, error)
	ConsumeQueue(queueName string) (<-chan Delivery, error)
	Close() error

	Ack(delivery *Delivery) error
	Nack(delivery *Delivery) error
}

// Backend -
type Backend interface {
	Close() error
	Connect() error
	GetChannel() (Channel, error)
	PurgeQueue(queueName string) error
}

// GetBackend -
func GetBackend(name string) (Backend, error) {
	switch name {
	case "redis":
		return NewREDIS(name), nil
	case "amqp":
		return NewAMQP(name), nil
	}
	return nil, fmt.Errorf("Invalid backend: %s", name)
}
