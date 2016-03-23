package main

import "github.com/streadway/amqp"

type AMQP struct {
	connectString string
	conn          *amqp.Connection
	channel       *amqp.Channel
}

func NewAMQP(connect string) *AMQP {
	/*
		Create a new AMQP object.
	*/
	server := &AMQP{
		connectString: connect,
	}
	return server
}

func (server *AMQP) Connect() error {
	/*
		Try to connect to the RabbitMQ server.
	*/
	conn, err := amqp.Dial(server.connectString)

	if err != nil {
		return err
	}

	server.conn = conn

	ch, err := server.conn.Channel()

	if err != nil {
		return err
	}

	server.channel = ch

	return nil
}

func (server *AMQP) Publish(queueName, body string) error {
	err := server.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	return err
}

func (server *AMQP) PurgeQueue(queueName string) error {
	/*
		Purge all pending messages in a queue.
	*/
	_, err := server.channel.QueuePurge(queueName, true)

	if err != nil {
		return err
	}
	return nil
}

func (server *AMQP) ConsumeQueue(queueName string) (<-chan amqp.Delivery, error) {
	/*
		Asynchronously consume items off the queue by returning a channel.
	*/

	q, err := server.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		return nil, err
	}

	msgs, err := server.channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (server *AMQP) Close() error {
	/*
		Close out the server when we are done.
	*/

	if server.conn != nil {
		server.conn.Close()
	}

	if server.channel != nil {
		server.channel.Close()
	}

	return nil
}
