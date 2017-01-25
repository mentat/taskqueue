package main

/*
  Client should set:

    SETNX task-name payload
    EXPIRE task-name tombstone-seconds
    RPUSH queue-name task-name

  Consumer should:

    RPOPLPUSH queue-name queue-name-lease

    or use the blocking version: BRPOPLPUSH

  On success consumer should:

    LREM queue-name-lease task-name

  On failure, consumer should pipeline:

    RPUSH queue-name task-name
    LREM queue-name-lease task-name

  Periodically check queue-name-lease to re-queue orphan items.
  Optimally we'd have some kind of lease time we'd check...


  Possibly do this on worker startup.

*/

import (
	"fmt"
	"time"

	"gopkg.in/redis.v5"
)

// REDIS -
type REDIS struct {
	connectString string
	client        *redis.Client
}

// REDISChannel -
type REDISChannel struct {
	name        string
	client      *redis.Client
	doneChannel chan int
}

// NewREDIS -Create a new Redis object.
func NewREDIS(connect string) *REDIS {

	server := &REDIS{
		connectString: connect,
	}
	return server
}

// Close - Close out the server when we are done.
func (server *REDIS) Close() error {

	if server.client != nil {
		server.client.Close()
	}

	return nil
}

// Connect - Try to connect to the RabbitMQ server.
func (server *REDIS) Connect() error {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	pong, err := client.Ping().Result()

	if err != nil {
		return err
	}

	server.client = client

	return nil
}

// GetChannel -
func (server *REDIS) GetChannel() (Channel, error) {
	/*pubsub, err := server.client.Subscribe("taskqueue.*")
	if err != nil {
		panic(err)
	}*/

	channel := &REDISChannel{
		client:      server.client,
		doneChannel: make(chan int),
	}

	return channel, nil
}

// PurgeQueue - Purge all pending messages in a queue.
func (server *REDIS) PurgeQueue(queueName string) error {
	server.client.LTrim(queueName, 0, -1)
	return nil
}

// Ack -
func (channel *REDISChannel) Ack(delivery *Delivery) error {

	err := channel.client.LRem(channel.name, 0, delivery.MessageID).Err()
	if err != nil {
		logger.Errorf("Error in Redis Ack: %s", err)
		return err
	}
	return nil
}

// Nack -
func (channel *REDISChannel) Nack(delivery *Delivery) error {

	pipe := channel.client.Pipeline()
	pipe.RPush(channel.name, 0, delivery.MessageID)
	pipe.LRem(fmt.Sprintf("%s.processing", channel.name), 0, delivery.MessageID)
	_, err := pipe.Exec()

	if err != nil {
		logger.Errorf("Error in Redis Nack: %s", err)
		return err
	}

	return nil
}

// Publish -
func (channel *REDISChannel) Publish(queueName, body string) error {
	err := channel.client.Publish(queueName, body).Err()
	if err != nil {
		logger.Errorf("Error publishing to Redis: %s", err)
		panic(err)
	}

	return err
}

// CountMessages - Get the count of pending messages in a queue.
func (channel *REDISChannel) CountMessages(queueName string) (int64, error) {

	return int64(0), nil
}

// ConsumeQueue - Asynchronously consume items off the queue by returning a channel.
func (channel *REDISChannel) ConsumeQueue(queueName string) (<-chan Delivery, error) {

	channel.name = queueName
	internalChannel := make(chan Delivery, 10)

	go func(done chan int) {
		for {
			select {
			case <-done:
				return
			default:
				msg := channel.client.BRPopLPush(
					queueName,
					fmt.Sprintf("%s.processing", queueName),
					time.Millisecond*200)

				name := msg.Val()

				payload := channel.client.Get(name)

				data, err := payload.Bytes()

				if err != nil {
					logger.Errorf("Cannot process value from Redis.")
					return
				}

				delivery := Delivery{
					MessageID: name,
					Body:      data,
				}
				internalChannel <- delivery
			}

		}

	}(channel.doneChannel)

	return internalChannel, nil
}

// Close - Close out the server when we are done.
func (channel *REDISChannel) Close() error {

	logger.Infof("Closing Redis channel.")
	channel.doneChannel <- 1

	return nil
}
