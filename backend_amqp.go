package main

import "github.com/streadway/amqp"

// AMQP -
type AMQP struct {
	connectString string
	conn          *amqp.Connection
}

// AMQPChannel -
type AMQPChannel struct {
	channel     *amqp.Channel
	doneChannel chan int
}

// NewAMQP -
func NewAMQP(connect string) *AMQP {
	/*
		Create a new AMQP object.
	*/
	server := &AMQP{
		connectString: connect,
	}
	return server
}

// Close -
func (server *AMQP) Close() error {
	/*
		Close out the server when we are done.
	*/

	if server.conn != nil {
		server.conn.Close()
	}

	return nil
}

// Connect -
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

// GetChannel -
func (server *AMQP) GetChannel() (Channel, error) {
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

func (channel *AMQPChannel) Ack(delivery *Delivery) error {
	return nil
}

func (channel *AMQPChannel) Nack(delivery *Delivery) error {
	return nil
}

func (channel *AMQPChannel) Publish(queueName, body string) error {
	err := channel.channel.Publish(
		"",        // exchange
		queueName, // routing key
		true,      // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	return err
}

func (channel *AMQPChannel) CountMessages(queueName string) (int64, error) {
	/*
		Get the count of pending messages in a queue.
	*/
	q, err := channel.channel.QueueInspect(queueName)

	if err != nil {
		return 0, err
	}
	return int64(q.Messages), nil
}

func (channel *AMQPChannel) ConsumeQueue(queueName string) (<-chan Delivery, error) {
	/*
		Asynchronously consume items off the queue by returning a channel.
	*/

	q, err := channel.channel.QueueDeclare(
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

	internalChannel := make(chan Delivery, 10)

	go func(done chan int) {

		msgs, err := channel.channel.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)

		if err != nil {
			return
		}

		select {
		case <-done:
			channel.channel.Close()
			return
		case in := <-msgs:
			delivery := Delivery{
				MessageID: in.MessageId,
				Body:      in.Body,
			}
			internalChannel <- delivery
		}

	}(channel.doneChannel)

	return internalChannel, nil
}

// Close -
func (channel *AMQPChannel) Close() error {
	/*
		Close out the server when we are done.
	*/
	logger.Infof("Closing AMQP channel.")
	channel.doneChannel <- 1

	return nil
}
