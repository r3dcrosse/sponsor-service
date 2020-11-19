package router

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/r3dcrosse/sponsor-service/common/db"
	"github.com/r3dcrosse/sponsor-service/common/messaging"
	"net/http"
	"strconv"
)

var MessagingClient messaging.IRabbitMQClient

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

func sendHttpErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(HttpErrorJSON{
		Success: false,
		Error: map[string]interface{}{
			"error": map[string]interface{}{
				"message": err.Error(),
			},
		},
	})
}

// To create a member of a sponsor team
func CreateMember(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])

	// Check if the event even exists
	event, err := db.GetEvent(eventId, -1)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}

	// Check if the sponsor team exists
	sponsorId, err := strconv.Atoi(params["sponsor_id"])
	s, err := db.GetSponsor(sponsorId)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}
	sponsor := Sponsor{
		Name: s.Name,
		Id:   s.ID,
	}

	// Get the sponsorship level from the DB
	l, err := db.GetLevel(s.LevelID)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
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
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
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
		err := MessagingClient.SendOnQueue(data, "sponsor.member.created")
		if err != nil {
			fmt.Printf("Something went wrong when sending the message to sponsor.member.created | %s", err.Error())
		}
	}(savedMember, eventId, level, sponsor)
}

// To create a level
func CreateLevel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])
	event, err := db.GetEvent(eventId, -1)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}

	level := Level{
		EventID: event.ID,
	}
	err = json.NewDecoder(r.Body).Decode(&level)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
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
func CreateSponsor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	eventId, err := strconv.Atoi(params["event_id"])
	event, err := db.GetEvent(eventId, -1)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}

	sponsor := Sponsor{
		Event: event.Name,
	}
	err = json.NewDecoder(r.Body).Decode(&sponsor)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
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
			sendHttpErrorResponse(w, http.StatusBadRequest, err)
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
}

// Get an event
func GetEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}

	result, err := db.GetEvent(id, -1)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
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
func GetAllEvents(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := db.GetAllEvents()

	var events []Event
	for _, result := range *results {
		var levels []Level
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

		var sponsors []Sponsor
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
			Name:     result.Name,
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
func CreateEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var event Event
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	// When we create an event in the DB, we pass in two vars here
	// First, the event name
	// Then, the Event service ID. We hard code -1 in this case
	// because events created through the REST API have no
	// corresponding ID from the event service, because they don't
	// exist in the event service
	result := db.CreateEvent(event.Name, -1)
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

func PatchEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r) // Gets params
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}

	var event Event
	err = json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	result, err := db.UpdateEvent(id, event.Name)
	if err != nil {
		sendHttpErrorResponse(w, http.StatusNotFound, err)
		return
	}
	savedEvent := Event{
		Id:   id,
		Name: result.Name,
	}

	// Check if the savedEvent has any levels
	if result.Levels != nil {
		for _, l := range result.Levels {
			savedEvent.Levels = append(savedEvent.Levels, Level{
				Id:                      l.ID,
				EventID:                 l.EventID,
				Name:                    l.Name,
				MaxSponsors:             l.MaxNumberOfSponsors,
				MaxFreeBadgesPerSponsor: l.MaxNumberOfFreeBadges,
				Cost:                    l.Cost,
			})
		}
	}

	// Check if event has levels to update
	if event.Levels != nil {
		for _, l := range event.Levels {
			var savedLevel *db.Level
			if l.Id == 0 {
				savedLevel = db.CreateLevel(l.Name, l.Cost, l.MaxSponsors, l.MaxFreeBadgesPerSponsor, id)
			} else {
				savedLevel, err = db.UpdateLevel(l.Id, l.Name, l.Cost, l.MaxSponsors, l.MaxFreeBadgesPerSponsor, id)
				if err != nil {
					sendHttpErrorResponse(w, http.StatusNotFound, err)
					return
				}
			}
			savedEvent.Levels = append(savedEvent.Levels, Level{
				Id:                      savedLevel.ID,
				Name:                    savedLevel.Name,
				Cost:                    savedLevel.Cost,
				MaxFreeBadgesPerSponsor: savedLevel.MaxNumberOfFreeBadges,
				MaxSponsors:             savedLevel.MaxNumberOfSponsors,
				EventID:                 savedLevel.EventID,
			})
		}
	}

	json.NewEncoder(w).Encode(HttpResponseJSON{
		Success: true,
		Data: map[string]interface{}{
			"event": savedEvent,
		},
	})
}

func RemoveMember(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r) // Gets params
	_, err := strconv.Atoi(params["event_id"])
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
		return
	}
	_, err = strconv.Atoi(params["sponsor_id"])
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	_, err = strconv.Atoi(params["member_id"])
	if err != nil {
		sendHttpErrorResponse(w, http.StatusBadRequest, err)
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
