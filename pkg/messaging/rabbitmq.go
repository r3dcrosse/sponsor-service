package messaging

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
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

// Initialize data
var Client RabbitMQClient

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// Function to connect to rabbit mq
func (m *RabbitMQClient) ConnectToRabbitMQ(ip string) {
	rabbitIP := fmt.Sprintf("amqp://guest:guest@%s/", ip)

	var err error
	m.connection, err = amqp.Dial(rabbitIP)
	if err != nil {
		failOnError(err, "Failed to connect to RabbitMQ at "+rabbitIP)
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
