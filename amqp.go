package main

import "github.com/streadway/amqp"

type AMQP struct {
	connectString string
	conn          *amqp.Connection
}

type AMQPChannel struct {
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

func (server *AMQP) Close() error {
	/*
		Close out the server when we are done.
	*/

	if server.conn != nil {
		server.conn.Close()
	}

	return nil
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

	return nil
}

func (server *AMQP) GetChannel() (*AMQPChannel, error) {
	ch, err := server.conn.Channel()

	if err != nil {
		return nil, err
	}

	channel := &AMQPChannel{
		channel: ch,
	}

	return channel, nil
}

func (server *AMQP) PurgeQueue(queueName string) error {
	/*
		Purge all pending messages in a queue.
	*/

	// Use a throw-away channel.
	ch, err := server.conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueuePurge(queueName, true)

	if err != nil {
		return err
	}

	return nil
}


func (server *AMQPChannel) Publish(queueName, body string) error {
	err := server.channel.Publish(
		"",        // exchange
		queueName, // routing key
		true,      // mandatory
		false,	   // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	return err
}

func (server *AMQPChannel) CountMessages(queueName string) (int64, error) {
	/*
		Get the count of pending messages in a queue.
	*/
	q, err := server.channel.QueueInspect(queueName)

	if err != nil {
		return 0, err
	}
	return int64(q.Messages), nil
}

func (server *AMQPChannel) ConsumeQueue(queueName string) (<-chan amqp.Delivery, error) {
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



func (server *AMQPChannel) Close() error {
	/*
		Close out the server when we are done.
	*/

	if server.channel != nil {
		server.channel.Close()
	}

	return nil
}
