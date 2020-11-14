package main

import (
	"encoding/json"
	"fmt"
	"flag"
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
	Event string `json:"event"`
	Level Level `json:"level"`
	Members []Members `json:"members"`
}

// Level struct
type Level struct {
	Name string `json:"name"`
	Cost string `json:"cost"`
	NumberOfBadges int `json:"number_of_badges"`
}

// Team Members struct (team members part of a sponsor)
type Members struct {
	Name string `json:"name"`
	Email string `json:"email"`
}

//////////////////////////////////////////////////////////////
//
// Our Microservice Models
//
//////////////////////////////////////////////////////////////
// Event struct
type Event struct {
	Name string
	Levels []Level
	Sponsors []Sponsor
}

// Initialize data
var events []Event
var sponsors []Sponsor

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

	sponsors = append(sponsors, sponsor)
	json.NewEncoder(w).Encode(sponsor)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func StartRabbitMQ(ip string) {
	rabbitIP := fmt.Sprintf("amqp://guest:guest@%s/", ip)

	conn, err := amqp.Dial(rabbitIP)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil, // arguments
	)
	failOnError(err, "Failed to declare a queue")

	body := "Hello World!"
	err = ch.Publish(
		"", // exchange
		q.Name, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing {
			ContentType: "text/plain",
			Body: []byte(body),
		})
	failOnError(err, "Failed to publish a message")
}

func main() {
	// Get any cmd line args passed to this service
	rabbitMQip := flag.String("rabbit", "localhost:5672", "IP Address and port where rabbitMQ is running")
	flag.Parse()

	// Initialize RabbitMQ
	StartRabbitMQ(*rabbitMQip)

	// Initialize the router
	router := mux.NewRouter()

	// Hardcoded data - @todo: add database

	// Route handles and endpoints
	router.HandleFunc("/sponsor/{event}", getSponsorsForEvent).Methods("GET") // show a list of sponsor organization names and each sponsor's level for an event
	router.HandleFunc("/sponsor-service/v1/sponsor", createSponsor).Methods("POST") // create a sponsor at a specific level
	//router.HandleFunc("/sponsor/{id}", updateSponsor).Methods("PUT") // add people on the sponsors team
	//router.HandleFunc("/sponsor/{id}", removeSponsor).Methods("DELETE") // remove people on the sponsors team

	// Start server
	log.Fatal(http.ListenAndServe(":8000", router))


}