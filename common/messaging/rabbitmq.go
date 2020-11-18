package messaging

import (
	"fmt"
	"github.com/r3dcrosse/sponsor-service/common/circuitbreaker"
	"github.com/streadway/amqp"
	"log"
	"time"
)

// RabbitMQ Interface for connecting, sending and receiving rabbit mq messages
type IRabbitMQClient interface {
	ConnectToRabbitMQ(rabbitMQip string)
	Send(msg []byte, exchangeName string, exchangeType string) error
	SendOnQueue(body []byte, queueName string) error
	Subscribe(exchangeName string, exchangeType string, consumerName string, handlerFunc func(delivery amqp.Delivery)) error
	SubscribeToQueue(queueName string, consumerName string, handlerFunc func(delivery amqp.Delivery)) error
	Close()
}

// Pointer to an amqp.Connection
type RabbitMQClient struct {
	connection *amqp.Connection
}

func (m *RabbitMQClient) Send(msg []byte, exchangeName string, exchangeType string) error {
	panic("implement me")
}

func (m *RabbitMQClient) Subscribe(exchangeName string, exchangeType string, consumerName string, handlerFunc func(delivery amqp.Delivery)) error {
	panic("implement me")
}

func (m *RabbitMQClient) Close() {
	panic("implement me")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// Function to connect to rabbit mq
func (m *RabbitMQClient) ConnectToRabbitMQ(ip string) {
	rabbitIP := fmt.Sprintf("amqp://guest:guest@%s/", ip)
	var err error

	////////////////////////////////////////////////////////////////
	// Attempt at Circuit breaker for connecting to rabbitmq
	//
	// Currently, this only works to handle going from:
	//   - Sponsor service starts, rabbitMQ has not started yet
	//   - Sponsor service waits for rabbitMQ to start
	//   - rabbitMQ has started
	//   - Sponsor service successfully connects
	////////////////////////////////////////////////////////////////
	for {
		if circuitbreaker.CB.Ready() {
			m.connection, err = amqp.Dial(rabbitIP)
			if err != nil {
				fmt.Printf("[%s] INFO: Could not find RabbitMQ at %s, will retry connecting | \n%s\n", time.Now(), rabbitIP, err.Error())
				circuitbreaker.CB.Fail()
				continue
			} else {
				fmt.Printf("[%s] INFO: Successfully connected to RabbitMQ at %s\n", time.Now(), rabbitIP)
				circuitbreaker.CB.Success()
				break
			}
		} else {
			// Breaker is in a tripped state
			//fmt.Printf("Failed to connect to RabbitMQ at %s, is it even up and running?\n", rabbitIP)
		}
	}
}

func (m *RabbitMQClient) SendOnQueue(body []byte, queueName string) error {
	ch, err := m.connection.Channel()
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Sends a message to the queue
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	failOnError(err, "Failed to publish a message")

	return err
}

func (m *RabbitMQClient) SubscribeToQueue(queueName string, consumerName string, handlerFunc func(delivery amqp.Delivery)) error {
	ch, err := m.connection.Channel()
	failOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to register a Queue")

	msgs, err := ch.Consume(
		q.Name,       // queue
		consumerName, // consumer
		true,         // auto ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	failOnError(err, "Failed to register a consumer")

	go consumeLoop(msgs, handlerFunc)
	return nil
}

func consumeLoop(deliveries <-chan amqp.Delivery, handlerFunc func(d amqp.Delivery)) {
	for d := range deliveries {
		handlerFunc(d)
	}
}
