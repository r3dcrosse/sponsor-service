package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//////////////////////////////////////////////////////////////
//
// Anti-Corruption Layer Models
//
//////////////////////////////////////////////////////////////
// Sponsor struct
type Sponsor struct {
	Event   string   `json:"event"`
	Level   Level    `json:"level"`
	Members []Member `json:"members"`
	Id      int      `json:"id"`
}

// Level struct
type Level struct {
	Name           string `json:"name"`
	Cost           string `json:"cost"`
	NumberOfBadges int    `json:"number_of_badges"`
}

// Team Member struct (team members part of a sponsor)
type Member struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Id    int    `json:"id"`
}

//////////////////////////////////////////////////////////////
//
// Our Microservice Models
//
//////////////////////////////////////////////////////////////
// Event struct
type Event struct {
	Name     string
	Levels   []Level
	Sponsors []Sponsor
}

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

// Initialize data
var events []Event
var sponsors []Sponsor

var messaging RabbitMQClient

// Get a list of sponsor organization names and each sponsor's level for an event
func getSponsorsForEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params

	// Looping through events to find the one from our request
	for _, event := range events {
		if event.Name == params["event"] {
			json.NewEncoder(w).Encode(event)
			return
		}
	}

	// Return an empty event if none is found
	json.NewEncoder(w).Encode(&Event{
		Name:     "",
		Levels:   nil,
		Sponsors: nil,
	})
}

// To create a member of a sponsor team
func createMember(m Member) Member {
	name := m.Name
	email := m.Email

	// @todo: create this member in the DB and get/set the ID for it
	id := 1337
	m.Id = id
	fmt.Printf("Trying to create member: %s with email: %s and id: %d\n", name, email, id)

	// Send a rabbitMQ message that a member was created
	go func(m Member) {
		memberNotification := Member{
			name,
			email,
			id,
		}
		data, _ := json.Marshal(memberNotification)
		err := messaging.SendOnQueue(data, "sponsor.member.created")
		if err != nil {
			failOnError(err, "Something went wrong when sending the message")
		}
	}(m)

	return Member(m)
}

// To create a sponsor
func createSponsor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var sponsor Sponsor
	err := json.NewDecoder(r.Body).Decode(&sponsor)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// @todo: create this sponsor in the DB and get/set the ID for it
	id := 1337
	sponsor.Id = id
	fmt.Printf("Trying to create sponsor for event: %s at level: %s and id: %d\n", sponsor.Event, sponsor.Level, id)

	// Check if we passed in any members of the sponsorship team
	var members []Member
	if len(sponsor.Members) > 0 {
		for _, member := range sponsor.Members {
			createdMember := createMember(member)
			members = append(members, createdMember)
		}

		sponsor.Members = members
	}

	sponsors = append(sponsors, sponsor)
	json.NewEncoder(w).Encode(sponsor)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	// Get any cmd line args passed to this service
	rabbitMQip := flag.String("rabbit", "localhost:5672", "IP Address and port where rabbitMQ is running")
	flag.Parse()

	// Initialize RabbitMQ
	messaging.ConnectToRabbitMQ(*rabbitMQip)
	//messaging.Subscribe("", "topic", "sponsor-service", "")

	// Initialize the router
	router := mux.NewRouter()

	// Hardcoded data - @todo: add database

	// Route handles and endpoints
	router.HandleFunc("/sponsor/{event}", getSponsorsForEvent).Methods("GET")       // show a list of sponsor organization names and each sponsor's level for an event
	router.HandleFunc("/sponsor-service/v1/sponsor", createSponsor).Methods("POST") // create a sponsor at a specific level
	//router.HandleFunc("/sponsor/{id}", updateSponsor).Methods("PUT") // add people on the sponsors team
	//router.HandleFunc("/sponsor/{id}", removeSponsor).Methods("DELETE") // remove people on the sponsors team

	// Start server
	log.Fatal(http.ListenAndServe(":8000", router))

}
