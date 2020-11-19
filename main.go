package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3dcrosse/sponsor-service/common/circuitbreaker"
	"github.com/r3dcrosse/sponsor-service/common/db"
	"github.com/r3dcrosse/sponsor-service/common/messaging"
	"github.com/r3dcrosse/sponsor-service/common/router"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"strings"
)

type LevelMessage struct {
	Name          string `json:"name"`
	Cost          int    `json:"cost"`
	MaxFreeBadges int    `json:"freeBadges"`
}

type EventMessage struct {
	Id            int            `json:"id"`
	Name          string         `json:"name"`
	SponsorLevels []LevelMessage `json:"sponsors"`
}

// Callback functions for everytime we get a message from rabbit mq
func onEventCreatedMessage(delivery amqp.Delivery) {
	msg := string(delivery.Body)

	// Format of the message will come in this shape:
	/*

			"
			EVENT CREATED ::: {
		      "id": 1337,
			  "name": "Super Awesome Event",
			  "sponsors": [
			    {
			      "name": "Platinum",
			      "cost": 14500,
			      "freeBadges": 10
			    }
			  ]
			}
			"

	*/

	// Split the delivery body on " ::: "
	s := strings.SplitAfter(msg, " ::: ")[1]
	dat := EventMessage{}
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		fmt.Printf("Could not parse new event json from rabbitmq message | %s", err)
		return
	}

	// Save the event in the DB
	savedEvent := db.CreateEvent(dat.Name, dat.Id)

	for _, l := range dat.SponsorLevels {
		level := router.Level{
			Name:                    l.Name,
			Cost:                    fmt.Sprintf("%d", l.Cost),
			MaxFreeBadgesPerSponsor: l.MaxFreeBadges,
		}

		db.CreateLevel(level.Name, level.Cost, level.MaxSponsors, level.MaxFreeBadgesPerSponsor, savedEvent.ID)
	}
}

func onEventModifiedMessage(delivery amqp.Delivery) {
	msg := string(delivery.Body)

	// Format of the message will come in this shape:
	/*

			"
			EVENT UPDATED ::: {
		      "id": 1337,
			  "name": "Super Awesome Event",
			  "sponsors": [
			    {
			      "name": "Platinum",
			      "cost": 14500,
			      "freeBadges": 10
			    }
			  ]
			}
			"

	*/

	// Split the delivery body on " ::: "
	s := strings.SplitAfter(msg, " ::: ")[1]
	dat := EventMessage{}
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		fmt.Printf("Could not parse new event json from rabbitmq message | %s", err.Error())
		return
	}

	// fetch the item in the DB
	// We send in -1 as our service ID since the event already
	// has an ID from the events service
	result, err := db.GetEvent(-1, dat.Id)
	if err != nil {
		fmt.Printf("Could not find the event to update based on the event ID | %s", err.Error())
		return
	}

	var levels []router.Level
	var sponsors []router.Sponsor
	event := router.Event{
		Id:       result.ID,
		Name:     dat.Name,
		Levels:   levels,
		Sponsors: sponsors,
	}

	if result.Levels != nil {
		for _, l := range result.Levels {
			levels = append(levels, router.Level{
				Id:                      l.ID,
				EventID:                 l.EventID,
				Cost:                    l.Cost,
				Name:                    l.Name,
				MaxFreeBadgesPerSponsor: l.MaxNumberOfFreeBadges,
				MaxSponsors:             l.MaxNumberOfSponsors,
			})
		}
	}

	if dat.SponsorLevels != nil {
		for _, l := range dat.SponsorLevels {
			level := router.Level{
				Name:                    l.Name,
				Cost:                    fmt.Sprintf("%d", l.Cost),
				MaxFreeBadgesPerSponsor: l.MaxFreeBadges,
			}

			result := db.CreateLevel(level.Name, level.Cost, level.MaxSponsors, level.MaxFreeBadgesPerSponsor, result.ID)
			level.Id = result.ID
			level.EventID = result.EventID
			level.MaxSponsors = result.MaxNumberOfSponsors

			levels = append(levels, level)
		}
	}

	db.UpdateEvent(result.ID, event.Name)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

var MessagingClient messaging.IRabbitMQClient

func main() {
	circuitbreaker.InitCircuitBreaker()

	// Get any cmd line args passed to this service
	rabbitMQip := flag.String("rabbit", "localhost:5672", "IP Address and port where rabbitMQ is running")
	postgresIp := flag.String("pg_ip", "localhost", "IP Address where postgres is running")
	postgresPort := flag.String("pg_port", "5432", "Port where postgres is running")
	postgresUser := flag.String("pg_user", "user", "User to use to login to postgres")
	postgresPass := flag.String("pg_password", "hey", "Password to use to login to postgres")
	postgresDbName := flag.String("pg_dbname", "postgres", "The db name to connect to")
	postgresSSL := flag.String("pg_ssl", "disable", "Run with ssl mode?")
	flag.Parse()

	// Initialize DB
	db.InitDB(db.Creds{
		Host:     *postgresIp,
		Port:     *postgresPort,
		User:     *postgresUser,
		Password: *postgresPass,
		Dbname:   *postgresDbName,
		Sslmode:  *postgresSSL,
	})

	// Initialize RabbitMQ
	MessagingClient = &messaging.RabbitMQClient{}
	MessagingClient.ConnectToRabbitMQ(*rabbitMQip)

	err := MessagingClient.SubscribeToQueue("event.create", "sponsor-service", onEventCreatedMessage)
	failOnError(err, "Could not subscribe to channel event.create")

	err = MessagingClient.SubscribeToQueue("event.modify", "sponsor-service", onEventModifiedMessage)
	failOnError(err, "Could not subscribe to channel event.modify")

	// Inject MessagingClient in router
	// So we can use rabbitMQ there if we get a request to create a new sponsor member
	router.MessagingClient = MessagingClient

	// Initialize the router
	r := mux.NewRouter()

	// Route handles and endpoints
	r.HandleFunc("/sponsor-service/v1/events", router.GetAllEvents).Methods("GET")
	r.HandleFunc("/sponsor-service/v1/event/{id}", router.GetEvent).Methods("GET")
	r.HandleFunc("/sponsor-service/v1/event", router.CreateEvent).Methods("POST")
	r.HandleFunc("/sponsor-service/v1/event/{id}", router.PatchEvent).Methods("PATCH")
	r.HandleFunc("/sponsor-service/v1/event/{event_id}/level", router.CreateLevel).Methods("POST")
	r.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor", router.CreateSponsor).Methods("POST")
	r.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member", router.CreateMember).Methods("POST")
	r.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member/{member_id}", router.RemoveMember).Methods("DELETE")

	// Start server
	log.Fatal(http.ListenAndServe(":8000", r))
}
