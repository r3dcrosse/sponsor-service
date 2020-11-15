package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/r3dcrosse/sponsor-service/pkg/messaging"
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
	EventID int      `json:"eventId"`
	Name    string   `json:"name"`
	Level   Level    `json:"level"`
	Members []Member `json:"members"`
	Id      int      `json:"id"`
}

// Level struct
type Level struct {
	EventID                 int    `json:"eventId"`
	Name                    string `json:"name"`
	Cost                    string `json:"cost"`
	MaxSponsors             int    `json:"maxSponsors"`
	MaxFreeBadgesPerSponsor int    `json:"maxFreeBadgesPerSponsor"`
	Id                      int    `json:"id"`
}

// Team Member struct (team members part of a sponsor)
type Member struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Id        int    `json:"id"`
	SponsorId int    `json:"sponsorId"`
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
func createMember(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])

	// Check if the event even exists
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

	// Check if the sponsor team exists
	sponsorId, err := strconv.Atoi(params["sponsor_id"])
	s, err := db.GetSponsor(sponsorId)
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
		Name: s.Name,
		Id:   s.ID,
	}

	// Get the sponsorship level from the DB
	l, err := db.GetLevel(s.LevelID)
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
	level := Level{
		Id:   l.ID,
		Name: l.Name,
	}

	member := Member{
		SponsorId: sponsorId,
	}
	err = json.NewDecoder(r.Body).Decode(&member)
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

	// Now create the member in the DB
	result := db.CreateMember(member.Name, member.Email, member.SponsorId)
	savedMember := Member{
		Id:        result.ID,
		Name:      result.Name,
		Email:     result.Email,
		SponsorId: result.SponsorID,
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"member": savedMember,
		},
	})

	// Send a rabbitMQ message that a member was created
	go func(m Member, evId int, l Level, s Sponsor) {
		memberNotification := map[string]interface{}{
			"id":           m.Id,
			"eventId":      evId,
			"sponsorId":    m.SponsorId,
			"name":         m.Name,
			"email":        m.Email,
			"organization": s.Name,
			"eventName":    event.Name,
			"sponsorLevel": l.Name,
		}
		data, _ := json.Marshal(memberNotification)
		err := messaging.Client.SendOnQueue(data, "sponsor.member.created")
		if err != nil {
			failOnError(err, "Something went wrong when sending the message")
		}
	}(savedMember, eventId, level, sponsor)

}

// To create a level
func createLevel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])
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

	level := Level{
		EventID: event.ID,
	}
	err = json.NewDecoder(r.Body).Decode(&level)
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

	result := db.CreateLevel(level.Name, level.Cost, level.MaxSponsors, level.MaxFreeBadgesPerSponsor, event.ID)
	savedLevel := Level{
		Id:                      result.ID,
		Name:                    result.Name,
		Cost:                    result.Cost,
		MaxFreeBadgesPerSponsor: result.MaxNumberOfFreeBadges,
		MaxSponsors:             result.MaxNumberOfSponsors,
		EventID:                 event.ID,
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"event": savedLevel,
		},
	})
}

// To create a sponsor
func createSponsor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])
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

	// Check for any Levels included in the request body
	level := Level{
		Id:                      sponsor.Level.Id,
		EventID:                 eventId,
		Name:                    sponsor.Level.Name,
		Cost:                    sponsor.Level.Cost,
		MaxSponsors:             sponsor.Level.MaxSponsors,
		MaxFreeBadgesPerSponsor: sponsor.Level.MaxFreeBadgesPerSponsor,
	}
	if sponsor.Level.Id != 0 {
		savedLevel, err := db.GetLevel(sponsor.Level.Id)
		// Check if the event IDs match...
		if err != nil || savedLevel.EventID != eventId {
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

		level.Id = savedLevel.ID
		level.EventID = savedLevel.EventID
		level.Name = savedLevel.Name
		level.Cost = savedLevel.Cost
		level.MaxSponsors = savedLevel.MaxNumberOfSponsors
		level.MaxFreeBadgesPerSponsor = savedLevel.MaxNumberOfFreeBadges
	} else {
		savedLevel := db.CreateLevel(level.Name, level.Cost, level.MaxSponsors, level.MaxFreeBadgesPerSponsor, eventId)
		level.Id = savedLevel.ID
		level.EventID = savedLevel.EventID
		level.Name = savedLevel.Name
		level.Cost = savedLevel.Cost
		level.MaxSponsors = savedLevel.MaxNumberOfSponsors
		level.MaxFreeBadgesPerSponsor = savedLevel.MaxNumberOfFreeBadges
	}

	if level.Id == 0 {
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
	} else {
		result := db.CreateSponsorWithLevel(sponsor.Name, level.Id, eventId)
		savedSponsor := Sponsor{
			Id:      result.ID,
			Name:    result.Name,
			Event:   event.Name,
			EventID: event.ID,
			Level:   level,
		}
		json.NewEncoder(w).Encode(HttpResponseJSON{
			Success: true,
			Data: map[string]interface{}{
				"sponsor": savedSponsor,
			},
		})
	}

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

// Get all events
// Get an event
func getAllEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := db.GetAllEvents()
	//if err != nil {
	//	w.WriteHeader(http.StatusBadRequest)
	//	json.NewEncoder(w).Encode(HttpErrorJSON{
	//		Success: false,
	//		Error: map[string]interface{}{
	//			"error": map[string]interface{}{
	//				"message": err.Error(),
	//			},
	//		},
	//	})
	//	return
	//}

	var events []Event
	for _, result := range *results {
		levels := []Level{}
		for _, level := range result.Levels {
			levels = append(levels, Level{
				EventID:                 level.EventID,
				Id:                      level.ID,
				MaxFreeBadgesPerSponsor: level.MaxNumberOfFreeBadges,
				MaxSponsors:             level.MaxNumberOfSponsors,
				Cost:                    level.Cost,
				Name:                    level.Name,
			})
		}

		sponsors := []Sponsor{}
		for _, sponsor := range result.Sponsors {
			sponsors = append(sponsors, Sponsor{
				Name:    sponsor.Name,
				Id:      sponsor.ID,
				EventID: sponsor.EventID,
				Level: Level{
					Name: sponsor.Level.Name,
				},
			})
		}

		events = append(events, Event{
			Id:       result.ID,
			Sponsors: sponsors,
			Levels:   levels,
		})
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"events": events,
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

func removeMember(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r) // Gets params
	_, err := strconv.Atoi(params["event_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"message": "Could not parse event ID from URL",
			},
		})
		return
	}
	_, err = strconv.Atoi(params["sponsor_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"message": "Could not parse the sponsor ID from URL",
			},
		})
		return
	}

	_, err = strconv.Atoi(params["member_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HttpErrorJSON{
			Success: false,
			Error: map[string]interface{}{
				"message": "Could not parse the member ID from URL",
			},
		})
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"message": "so sorry, this isn't implemented yet...",
		},
	})
}

// Callback functions for everytime we get a message from rabbit mq
func onEventCreatedMessage(delivery amqp.Delivery) {
	// @todo: Implement handling parsing message and creating an event from the message
	fmt.Printf("Got this message from event.create: %v\n", string(delivery.Body))
}

func onEventModifiedMessage(delivery amqp.Delivery) {
	// @todo: Implement handling parsing message and modifying an event from the message
	fmt.Printf("Got this message from event.modify: %v\n", string(delivery.Body))
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	// Get any cmd line args passed to this service
	rabbitMQip := flag.String("rabbit", "localhost:5672", "IP Address and port where rabbitMQ is running")
	postgresIp := flag.String("pg_ip", "localhost", "IP Address where postgres is running")
	postgresPort := flag.String("pg_port", "5432", "Port where postgres is running")
	postgresUser := flag.String("pg_user", "user", "User to use to login to postgres")
	postgresPass := flag.String("pg_password", "hey", "Password to use to login to postgres")
	postgresDbName := flag.String("pg_dbname", "postgres", "The db name to connect to")
	postgresSSL := flag.String("pg_ssl", "disable", "Run with ssl mode?")
	flag.Parse()

	// Initialize RabbitMQ
	messaging.Client.ConnectToRabbitMQ(*rabbitMQip)
	err := messaging.Client.SubscribeToQueue("event.create", "sponsor-service", onEventCreatedMessage)
	failOnError(err, "Could not subscribe to channel event.create")

	err = messaging.Client.SubscribeToQueue("event.modify", "sponsor-service", onEventModifiedMessage)
	failOnError(err, "Could not subscribe to channel event.modify")

	//initializeDb()
	db.InitDB(db.Creds{
		Host:     *postgresIp,
		Port:     *postgresPort,
		User:     *postgresUser,
		Password: *postgresPass,
		Dbname:   *postgresDbName,
		Sslmode:  *postgresSSL,
	})

	// Initialize the router
	router := mux.NewRouter()

	// Route handles and endpoints
	router.HandleFunc("/sponsor-service/v1/events", getAllEvents).Methods("GET")
	router.HandleFunc("/sponsor-service/v1/event/{id}", getEvent).Methods("GET")
	router.HandleFunc("/sponsor-service/v1/event", createEvent).Methods("POST")
	router.HandleFunc("/sponsor-service/v1/event/{event_id}/level", createLevel).Methods("POST")
	//router.HandleFunc("/sponsor/{event}", getSponsorsForEvent).Methods("GET")       // show a list of sponsor organization names and each sponsor's level for an event
	router.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor", createSponsor).Methods("POST") // create a sponsor at a specific level
	router.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member", createMember).Methods("POST")
	router.HandleFunc("/sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member/{member_id}", removeMember).Methods("DELETE")

	// Start server
	log.Fatal(http.ListenAndServe(":8000", router))

}
