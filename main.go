package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/r3dcrosse/sponsor-service/pkg/db"
)

//////////////////////////////////////////////////////////////
//
// Our Microservice Models
//
//////////////////////////////////////////////////////////////
// Sponsor struct
type Sponsor struct {
	Event   string   `json:"event"`
	EventID int      `json:"event_id"`
	Name    string   `json:"name"`
	Level   Level    `json:"level"`
	Members []Member `json:"members"`
	Id      int      `json:"id"`
}

// Level struct
type Level struct {
	Name           string `json:"name"`
	Cost           string `json:"cost"`
	NumberOfBadges int    `json:"number_of_badges"`
	Id             int    `json:"id"`
}

// Team Member struct (team members part of a sponsor)
type Member struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Id    int    `json:"id"`
}

//////////////////////////////////////////////////////////////
//
// Anti-Corruption Layer Models
//
//////////////////////////////////////////////////////////////
// Event struct
type Event struct {
	Id       int       `json:"id"`
	Name     string    `json:"name"`
	Levels   []Level   `json:"levels"`
	Sponsors []Sponsor `json:"sponsors"`
}

// HttpResponse JSON struct
type HttpResponseJSON struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

// HttpError JSON struct
type HttpErrorJSON struct {
	Success bool                   `json:"success"`
	Error   map[string]interface{} `json:"error"`
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
var messaging RabbitMQClient

// Get a list of sponsor organization names and each sponsor's level for an event
//func getSponsorsForEvent(w http.ResponseWriter, r *http.Request) {
//	w.Header().Set("Content-Type", "application/json")
//	params := mux.Vars(r) // Gets params
//
//	// Looping through events to find the one from our request
//	for _, event := range events {
//		if event.Name == params["event"] {
//			json.NewEncoder(w).Encode(event)
//			return
//		}
//	}
//
//	// Return an empty event if none is found
//	json.NewEncoder(w).Encode(&Event{
//		Name:     "",
//		Levels:   nil,
//		Sponsors: nil,
//	})
//}

// To create a member of a sponsor team
func createMember(m Member) Member {
	name := m.Name
	email := m.Email

	// @todo: create this member in the DB and get/set the ID for it
	db.CreateMember(struct {
		Name  string
		Email string
	}{Name: m.Name, Email: m.Email})
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
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])
	fmt.Printf("USING EVENT ID: %d", eventId)
	event, err := db.GetEvent(eventId)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"error": map[string]interface{}{
					"message": err.Error(),
				},
			},
		})
		return
	}

	sponsor := Sponsor{
		Event: event.Name,
	}
	err = json.NewDecoder(r.Body).Decode(&sponsor)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"error": map[string]interface{}{
					"message": err.Error(),
				},
			},
		})
		return
	}

	result := db.CreateSponsor(sponsor.Name, event.ID)
	savedSponsor := Sponsor{
		Id:      result.ID,
		Name:    result.Name,
		Event:   event.Name,
		EventID: event.ID,
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"sponsor": savedSponsor,
		},
	})

	// Check if we passed in any members of the sponsorship team
	//var members []Member
	//if len(sponsor.Members) > 0 {
	//	for _, member := range sponsor.Members {
	//		createdMember := createMember(member)
	//		members = append(members, createdMember)
	//	}
	//
	//	sponsor.Members = members
	//}
	//
	//sponsors = append(sponsors, sponsor)
	//json.NewEncoder(w).Encode(sponsor)
}

// Get an event
func getEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		failOnError(err, "Could not parse event ID from URL")
	}

	result, err := db.GetEvent(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"error": map[string]interface{}{
					"message": err.Error(),
				},
			},
		})
		return
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"event": &Event{
				Id:   result.ID,
				Name: result.Name,
			},
		},
	})
}

// To create an event
func createEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var event Event
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := db.CreateEvent(event.Name)
	savedEvent := Event{
		Id:   result.ID,
		Name: result.Name,
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"event": savedEvent,
		},
	})
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

	//initializeDb()
	db.InitDB()

	// Initialize the router
	router := mux.NewRouter()

	// Route handles and endpoints
	router.HandleFunc("/sponsor-service/v1/event/{id}", getEvent).Methods("GET")
	router.HandleFunc("/sponsor-service/v1/event", createEvent).Methods("POST")
	//router.HandleFunc("/sponsor/{event}", getSponsorsForEvent).Methods("GET")       // show a list of sponsor organization names and each sponsor's level for an event
	router.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor", createSponsor).Methods("POST") // create a sponsor at a specific level
	//router.HandleFunc("/sponsor/{id}", updateSponsor).Methods("PUT") // add people on the sponsors team
	//router.HandleFunc("/sponsor/{id}", removeSponsor).Methods("DELETE") // remove people on the sponsors team

	// Start server
	log.Fatal(http.ListenAndServe(":8000", router))

}
